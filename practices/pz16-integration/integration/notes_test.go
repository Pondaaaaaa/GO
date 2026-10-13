package integration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/db"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/httpapi"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/models"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/repo"
	"github.com/Pondaaaaaa/GO/practices/pz16-integration/internal/service"
)

type fixture struct {
	db     *sql.DB
	server *httptest.Server
	client *http.Client
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Fatal("DB_DSN missing: run run-local.ps1; integration tests must not skip")
	}
	project := os.Getenv("PZ16_PROJECT")
	if !strings.HasPrefix(project, "pz16-") || len(project) != 37 {
		t.Fatal("missing owned Compose project marker")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.Hostname() != "127.0.0.1" || u.Path != "/notes_test" {
		t.Fatal("only owned loopback notes_test database is accepted")
	}
	admin, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal("test database connection failed (credentials omitted)")
	}
	t.Cleanup(func() { _ = admin.Close() })
	var database string
	if err = admin.QueryRow("SELECT current_database()").Scan(&database); err != nil || database != "notes_test" {
		t.Fatal("unexpected database")
	}
	var token [12]byte
	if _, err = rand.Read(token[:]); err != nil {
		t.Fatal("random schema token failed")
	}
	schema := "it_" + hex.EncodeToString(token[:]) // generated identifier, never supplied by user
	if _, err = admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal("create isolated schema failed")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("isolated schema cleanup failed")
		}
	})
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	d, err := db.Open(context.Background(), u.String())
	if err != nil {
		t.Fatal("schema database connection failed")
	}
	t.Cleanup(func() { _ = d.Close() })
	if db.Migrate(context.Background(), d) != nil {
		t.Fatal("migration failed")
	}
	srv := httptest.NewServer(httpapi.New(service.Service{Notes: repo.Notes{DB: d}}))
	t.Cleanup(srv.Close)
	return &fixture{db: d, server: srv, client: &http.Client{Timeout: 5 * time.Second}}
}

func (f *fixture) request(t *testing.T, method, path, body string, want int) ([]byte, http.Header) {
	t.Helper()
	return f.requestType(t, method, path, body, "application/json", want)
}
func (f *fixture) requestType(t *testing.T, method, path, body, contentType string, want int) ([]byte, http.Header) {
	t.Helper()
	r, err := http.NewRequest(method, f.server.URL+"/api/v1/notes"+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", contentType)
	resp, err := f.client.Do(r)
	if err != nil {
		t.Fatal("HTTP request failed")
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, resp.StatusCode, want, b)
	}
	if want == 204 {
		if len(b) != 0 || resp.Header.Get("Content-Type") != "" {
			t.Fatal("204 must have no body or JSON header")
		}
	} else {
		media, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if err != nil || media != "application/json" || !json.Valid(b) {
			t.Fatalf("invalid JSON response: %q %s", resp.Header.Get("Content-Type"), b)
		}
	}
	if want >= 400 {
		var out map[string]string
		if json.Unmarshal(b, &out) != nil || len(out) != 1 {
			t.Fatal("invalid error object")
		}
		expected := "invalid request"
		if want == 404 {
			expected = "note not found"
		}
		if out["error"] != expected {
			t.Fatalf("error=%q want=%q", out["error"], expected)
		}
	}
	if len(b) < 512 {
		t.Logf("%s /api/v1/notes%s -> %d Content-Type=%q %s", method, path, want, resp.Header.Get("Content-Type"), strings.TrimSpace(string(b)))
	}
	return b, resp.Header
}
func note(t *testing.T, b []byte) models.Note {
	t.Helper()
	var n models.Note
	if json.Unmarshal(b, &n) != nil || n.ID <= 0 || n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
		t.Fatalf("invalid note: %s", b)
	}
	return n
}
func (f *fixture) create(t *testing.T) models.Note {
	t.Helper()
	b, h := f.request(t, "POST", "", `{"title":"  Hello  ","content":"World"}`, 201)
	n := note(t, b)
	if n.Title != "Hello" || n.Content != "World" || h.Get("Location") != fmt.Sprintf("/api/v1/notes/%d", n.ID) {
		t.Fatalf("wrong create result: %+v", n)
	}
	return n
}
func (f *fixture) stored(t *testing.T, id int64) models.Note {
	t.Helper()
	var n models.Note
	err := f.db.QueryRow("SELECT id,title,content,created_at,updated_at FROM notes WHERE id=$1", id).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		t.Fatal("expected stored note missing")
	}
	return n
}
func equalNote(t *testing.T, a, b models.Note) {
	t.Helper()
	if a.ID != b.ID || a.Title != b.Title || a.Content != b.Content || !a.CreatedAt.Equal(b.CreatedAt) || !a.UpdatedAt.Equal(b.UpdatedAt) {
		t.Fatalf("note mismatch: %+v versus %+v", a, b)
	}
}
func (f *fixture) count(t *testing.T) int {
	t.Helper()
	var count int
	if f.db.QueryRow("SELECT count(*) FROM notes").Scan(&count) != nil {
		t.Fatal("count failed")
	}
	return count
}

func TestCreateAndGetNote(t *testing.T) {
	f := newFixture(t)
	n := f.create(t)
	equalNote(t, n, f.stored(t, n.ID))
	b, _ := f.request(t, "GET", fmt.Sprintf("/%d", n.ID), "", 200)
	equalNote(t, n, note(t, b))
	if f.count(t) != 1 {
		t.Fatal("INSERT count must be one")
	}
	t.Log("DB INSERT verified: title=Hello content=World count=1")
}
func TestGetMissingNote(t *testing.T) {
	f := newFixture(t)
	f.request(t, "GET", "/999999", "", 404)
	if f.count(t) != 0 {
		t.Fatal("missing GET changed DB")
	}
}
func TestUpdateNote(t *testing.T) {
	f := newFixture(t)
	before := f.create(t)
	path := fmt.Sprintf("/%d", before.ID)
	b, _ := f.request(t, "PATCH", path, `{"title":"  Changed  "}`, 200)
	after := note(t, b)
	if after.Title != "Changed" || after.Content != "World" || !after.CreatedAt.Equal(before.CreatedAt) || !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("invalid update: %+v", after)
	}
	equalNote(t, after, f.stored(t, before.ID))
	b, _ = f.request(t, "PATCH", path, `{"content":""}`, 200)
	final := note(t, b)
	if final.Content != "" || final.Title != "Changed" {
		t.Fatal("empty content PATCH failed")
	}
	equalNote(t, final, f.stored(t, before.ID))
	f.request(t, "PATCH", "/999999", `{"content":"x"}`, 404)
	equalNote(t, final, f.stored(t, before.ID))
	if f.count(t) != 1 {
		t.Fatal("missing PATCH changed row count")
	}
	t.Log("DB UPDATE verified: partial fields preserved, timestamps advanced, empty content allowed")
}
func TestDeleteNote(t *testing.T) {
	f := newFixture(t)
	n := f.create(t)
	path := fmt.Sprintf("/%d", n.ID)
	f.request(t, "DELETE", path, "", 204)
	if f.count(t) != 0 {
		t.Fatal("DELETE left a row")
	}
	f.request(t, "GET", path, "", 404)
	f.request(t, "DELETE", path, "", 404)
	t.Log("DB DELETE verified: count=0; 204 has empty body")
}
func TestListNotes(t *testing.T) {
	f := newFixture(t)
	b, _ := f.request(t, "GET", "", "", 200)
	if strings.TrimSpace(string(b)) != "[]" {
		t.Fatal("empty list must be []")
	}
	a := f.create(t)
	b, _ = f.request(t, "POST", "", `{"title":"Second","content":"Two"}`, 201)
	second := note(t, b)
	b, _ = f.request(t, "GET", "", "", 200)
	var list []models.Note
	if json.Unmarshal(b, &list) != nil || len(list) != 2 {
		t.Fatal("list must contain two notes")
	}
	equalNote(t, list[0], a)
	equalNote(t, list[1], second)
	if f.count(t) != 2 {
		t.Fatal("list changed DB count")
	}
	t.Log("DB list verified: stable ID order and empty array")
}
func TestInvalidRequests(t *testing.T) {
	f := newFixture(t)
	n := f.create(t)
	path := fmt.Sprintf("/%d", n.ID)
	cases := []struct{ name, method, path, body, ct string }{
		{"malformed", "POST", "", `{`, "application/json"},
		{"blank_title", "POST", "", `{"title":"   "}`, "application/json"},
		{"unknown_field", "POST", "", `{"title":"x","extra":1}`, "application/json"},
		{"trailing_json", "POST", "", `{"title":"x"}{}`, "application/json"},
		{"wrong_content_type", "POST", "", `{"title":"x"}`, "text/plain"},
		{"fake_json_type", "POST", "", `{"title":"x"}`, "application/jsonfoo"},
		{"null_create", "POST", "", `null`, "application/json"},
		{"bad_id", "GET", "/abc", "", "application/json"},
		{"zero_id", "GET", "/0", "", "application/json"},
		{"negative_id", "DELETE", "/-1", "", "application/json"},
		{"overflow_id", "GET", "/9223372036854775808", "", "application/json"},
		{"empty_patch", "PATCH", path, `{}`, "application/json"},
		{"null_patch", "PATCH", path, `{"title":null}`, "application/json"},
		{"blank_patch", "PATCH", path, `{"title":" "}`, "application/json"},
		{"bad_patch_id", "PATCH", "/bad", `{"title":"x"}`, "application/json"},
		{"oversized", "POST", "", `{"title":"x","content":"` + strings.Repeat("x", 1<<20) + `"}`, "application/json"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f.requestType(t, c.method, c.path, c.body, c.ct, 400)
			if f.count(t) != 1 {
				t.Fatal("invalid request changed row count")
			}
			equalNote(t, n, f.stored(t, n.ID))
		})
	}
	t.Log("DB unchanged after all invalid requests")
}
func TestMigrationIdempotent(t *testing.T) {
	f := newFixture(t)
	n := f.create(t)
	if db.Migrate(context.Background(), f.db) != nil || db.Migrate(context.Background(), f.db) != nil {
		t.Fatal("repeated migration failed")
	}
	equalNote(t, n, f.stored(t, n.ID))
	if f.count(t) != 1 {
		t.Fatal("migration damaged data")
	}
	_, err := f.db.Exec("INSERT INTO notes(title,content) VALUES('   ','x')")
	if err == nil {
		t.Fatal("blank title constraint missing")
	}
	_, err = f.db.Exec("INSERT INTO notes(title,content) VALUES(NULL,'x')")
	if err == nil {
		t.Fatal("NOT NULL constraint missing")
	}
	if f.count(t) != 1 {
		t.Fatal("constraint failures inserted data")
	}
	t.Log("Migration repeated twice: existing row unchanged; CHECK and NOT NULL enforced")
}
func TestRepositoryCanceledContext(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (repo.Notes{DB: f.db}).Create(ctx, models.Create{Title: "Canceled"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
	if f.count(t) != 0 {
		t.Fatal("canceled INSERT changed DB")
	}
}
