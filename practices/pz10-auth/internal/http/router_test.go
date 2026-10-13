package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/core"
	jwtutil "github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/platform/jwt"
	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/repo"
)

func TestAuthFlow(t *testing.T) {
	users, err := repo.NewUserMem()
	if err != nil {
		t.Fatal(err)
	}
	manager := jwtutil.New([]byte(strings.Repeat("s", 32)), time.Minute, time.Hour)
	app := Build(&Service{Users: users, Tokens: manager, Blacklist: core.NewRefreshBlacklist()})
	call := func(method, path string, body any, token string) (int, map[string]any) {
		t.Helper()
		var payload []byte
		if body != nil {
			payload, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("missing JSON type for %s", path)
		}
		return rec.Code, out
	}
	if code, _ := call("GET", "/api/v1/me", nil, ""); code != 401 {
		t.Fatalf("missing token: %d", code)
	}
	code, admin := call("POST", "/api/v1/login", map[string]string{"email": "admin@example.com", "password": "secret123"}, "")
	if code != 200 {
		t.Fatalf("admin login: %d %v", code, admin)
	}
	access := admin["access_token"].(string)
	refresh := admin["refresh_token"].(string)
	if code, _ := call("GET", "/api/v1/me", nil, access); code != 200 {
		t.Fatalf("me: %d", code)
	}
	if code, _ := call("GET", "/api/v1/admin/stats", nil, access); code != 200 {
		t.Fatalf("admin: %d", code)
	}
	if code, _ := call("GET", "/api/v1/me", nil, refresh); code != 401 {
		t.Fatalf("refresh used as access: %d", code)
	}
	code, rotated := call("POST", "/api/v1/refresh", map[string]string{"refresh_token": refresh}, "")
	if code != 200 {
		t.Fatalf("refresh: %d %v", code, rotated)
	}
	if rotated["access_token"] == access {
		t.Fatal("access token did not rotate")
	}
	if code, _ := call("POST", "/api/v1/refresh", map[string]string{"refresh_token": refresh}, ""); code != 401 {
		t.Fatalf("refresh replay: %d", code)
	}
	code, user := call("POST", "/api/v1/login", map[string]string{"email": "user@example.com", "password": "secret123"}, "")
	if code != 200 {
		t.Fatalf("user login: %d", code)
	}
	userAccess := user["access_token"].(string)
	if code, _ := call("GET", "/api/v1/admin/stats", nil, userAccess); code != 403 {
		t.Fatalf("user admin access: %d", code)
	}
	if code, _ := call("GET", "/api/v1/users/2", nil, userAccess); code != 200 {
		t.Fatalf("own profile: %d", code)
	}
	if code, _ := call("GET", "/api/v1/users/1", nil, userAccess); code != 403 {
		t.Fatalf("foreign profile: %d", code)
	}
	if code, _ := call("GET", "/api/v1/users/"+strconv.Itoa(2), nil, access); code != 200 {
		t.Fatalf("admin profile: %d", code)
	}
	if code, _ := call("POST", "/api/v1/login", map[string]string{"email": "user@example.com", "password": "wrong"}, ""); code != 401 {
		t.Fatalf("bad password: %d", code)
	}
	if code, _ := call("POST", "/api/v1/refresh", map[string]string{"refresh_token": access}, ""); code != 401 {
		t.Fatalf("access used as refresh: %d", code)
	}
}
