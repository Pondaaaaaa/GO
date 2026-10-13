package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz14-notes-db/internal/notes"
)

var run = flag.String("run", "", "32 lowercase hex run ID")
var action = flag.String("action", "prepare", "prepare/plans/check/load")
var address = flag.String("url", "", "API URL")
var optimized = flag.Bool("optimized", false, "optimized workload")
var hexID = regexp.MustCompile(`^[a-f0-9]{32}$`)
var cursor = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(20001 * time.Second)

func main() {
	flag.Parse()
	if !hexID.MatchString(*run) {
		log.Fatal("invalid run identifier")
	}
	if *action == "load" {
		if err := load(); err != nil {
			log.Fatal(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := notes.Open(ctx, 8)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	switch *action {
	case "prepare":
		err = prepare(ctx, db)
	case "plans":
		err = plans(ctx, db)
	case "check":
		err = check(ctx, db)
	default:
		err = fmt.Errorf("invalid action")
	}
	if err != nil {
		log.Fatal("lab operation failed (credentials omitted)")
	}
}

func prepare(ctx context.Context, db *sql.DB) error {
	var version string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&version); err != nil {
		return err
	}
	fmt.Println("PostgreSQL server_version=" + version)
	for _, kind := range []string{"base", "opt"} {
		s := notes.Schema(*run, kind)
		// A new run owns new schemas: never drop an index or modify old rows.
		ddl := "CREATE SCHEMA " + s + "; CREATE TABLE " + s + ".notes(id BIGSERIAL PRIMARY KEY,title TEXT NOT NULL,content TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),updated_at TIMESTAMPTZ); CREATE TABLE " + s + ".note_audit(id BIGSERIAL PRIMARY KEY,note_id BIGINT NOT NULL,action TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now());"
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO "+s+".notes(id,title,content,created_at) SELECT n, CASE WHEN n%1000=0 THEN 'redis ' ELSE 'note ' END||n::text,repeat('content ',20),TIMESTAMPTZ '2026-01-01 00:00:00+00'+n*INTERVAL '1 second' FROM generate_series(1,100000) n"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "SELECT setval('"+s+".notes_id_seq',100000)"); err != nil {
			return err
		}
		if kind == "opt" {
			if _, err := db.ExecContext(ctx, "CREATE INDEX notes_created_id ON "+s+".notes(created_at DESC,id DESC); CREATE INDEX notes_title_fts ON "+s+".notes USING GIN(to_tsvector('simple',title))"); err != nil {
				return err
			}
		}
		if _, err := db.ExecContext(ctx, "ANALYZE "+s+".notes"); err != nil {
			return err
		}
		fmt.Printf("%s: 100000 deterministic rows, primary key%s\n", kind, map[bool]string{true: " plus created/id and full-text indexes", false: " only"}[kind == "opt"])
	}
	var preload string
	if err := db.QueryRowContext(ctx, "SHOW shared_preload_libraries").Scan(&preload); err != nil {
		return err
	}
	if !strings.Contains(preload, "pg_stat_statements") {
		fmt.Println("pg_stat_statements unavailable: absent from shared_preload_libraries; global configuration unchanged.")
		return nil
	}
	if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements"); err != nil {
		fmt.Println("pg_stat_statements unavailable: extension creation denied; global configuration unchanged.")
		return nil
	}
	fmt.Println("pg_stat_statements extension available (total_exec_time/mean_exec_time)")
	return nil
}
func explain(ctx context.Context, db *sql.DB, label, query string, args ...any) error {
	fmt.Println("PLAN " + label)
	fmt.Println("SQL: " + query)
	rows, err := db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var line string
		if err = rows.Scan(&line); err != nil {
			return err
		}
		fmt.Println(line)
	}
	return rows.Err()
}
func plans(ctx context.Context, db *sql.DB) error {
	for _, kind := range []string{"base", "opt"} {
		t := notes.Schema(*run, kind) + ".notes"
		if kind == "base" {
			if err := explain(ctx, db, "baseline OFFSET 80000", "SELECT id,title,content,created_at,updated_at FROM "+t+" ORDER BY created_at DESC,id DESC OFFSET $1 LIMIT $2", 80000, 50); err != nil {
				return err
			}
		} else {
			if err := explain(ctx, db, "optimized keyset", "SELECT id,title,content,created_at,updated_at FROM "+t+" WHERE (created_at,id)<($1,$2) ORDER BY created_at DESC,id DESC LIMIT $3", cursor, 20001, 50); err != nil {
				return err
			}
		}
		if err := explain(ctx, db, kind+" full-text redis", "SELECT id,title FROM "+t+" WHERE to_tsvector('simple',title) @@ plainto_tsquery('simple',$1)", "redis"); err != nil {
			return err
		}
		if kind == "base" {
			if err := explain(ctx, db, "N+1 representative point query, repeated 20 times by API", "SELECT id,title,content,created_at,updated_at FROM "+t+" WHERE id=$1", int64(1)); err != nil {
				return err
			}
		} else {
			if err := explain(ctx, db, "batch 20 IDs", "SELECT id,title,content,created_at,updated_at FROM "+t+" WHERE id=ANY($1::bigint[]) ORDER BY array_position($1::bigint[],id)", ids20()); err != nil {
				return err
			}
		}
	}
	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_stat_statements')").Scan(&exists); err != nil {
		return err
	}
	if exists {
		rows, err := db.QueryContext(ctx, "SELECT calls,total_exec_time,mean_exec_time,rows FROM pg_stat_statements WHERE query LIKE $1 ORDER BY total_exec_time DESC LIMIT 10", "%pz14_"+*run+"%")
		if err != nil {
			fmt.Println("pg_stat_statements view not usable; relying on EXPLAIN/app metrics.")
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var calls, n int64
			var total, mean float64
			rows.Scan(&calls, &total, &mean, &n)
			fmt.Printf("pg_stat_statements calls=%d total_exec_time_ms=%.3f mean_exec_time_ms=%.3f rows=%d\n", calls, total, mean, n)
		}
	}
	return nil
}
func ids20() []int64 {
	out := make([]int64, 20)
	for i := range out {
		out[i] = int64(i + 1)
	}
	return out
}
func check(ctx context.Context, db *sql.DB) error {
	base, err := notes.New(ctx, db, notes.Schema(*run, "base"), false)
	if err != nil {
		return err
	}
	defer base.Close()
	opt, err := notes.New(ctx, db, notes.Schema(*run, "opt"), true)
	if err != nil {
		return err
	}
	defer opt.Close()
	a, err := base.List(ctx, 50, 80000, nil, 0)
	if err != nil {
		return err
	}
	b, err := opt.List(ctx, 50, 0, &cursor, 20001)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("pagination mismatch")
	}
	fmt.Println("OFFSET/keyset: same 50 rows at logical offset 80000")
	a, err = base.Batch(ctx, ids20())
	if err != nil {
		return err
	}
	b, err = opt.Batch(ctx, ids20())
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("batch mismatch")
	}
	fmt.Println("N+1/batch: same 20 rows in requested order")
	var seen = map[int64]bool{}
	var ct *time.Time
	var id int64
	for page := 0; page < 3; page++ {
		rows, err := opt.List(ctx, 100, 0, ct, id)
		if err != nil {
			return err
		}
		if len(rows) != 100 {
			return fmt.Errorf("short page")
		}
		for i, n := range rows {
			expected := int64(100000 - page*100 - i)
			if n.ID != expected || seen[n.ID] {
				return fmt.Errorf("keyset missing/duplicate")
			}
			seen[n.ID] = true
		}
		last := rows[len(rows)-1]
		ct = &last.CreatedAt
		id = last.ID
	}
	fmt.Println("keyset walk: 300 consecutive rows, no missing IDs or duplicates")
	var before, after int64
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+opt.Table).Scan(&before); err != nil {
		return err
	}
	_, err = opt.CreateMany(ctx, []notes.Input{{Title: "rollback probe"}, {Title: " "}})
	if err != notes.ErrInvalid {
		return fmt.Errorf("expected transactional failure")
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+opt.Table).Scan(&after); err != nil {
		return err
	}
	if before != after {
		return fmt.Errorf("rollback failed")
	}
	fmt.Println("transaction rollback: first insert absent after invalid second item")
	created, err := opt.CreateMany(ctx, []notes.Input{{Title: "transaction A"}, {Title: "transaction B"}})
	if err != nil {
		return err
	}
	var audits int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM "+opt.Audit+" WHERE note_id=ANY($1::bigint[])", []int64{created[0].ID, created[1].ID}).Scan(&audits)
	if err != nil || audits != 2 {
		return fmt.Errorf("audit mismatch")
	}
	fmt.Println("transaction commit: two notes and two matching audit entries")
	for _, n := range created {
		if err = opt.Delete(ctx, n.ID); err != nil {
			return err
		}
	}
	return nil
}

func getJSON(client *http.Client, path string, v any) error {
	resp, err := client.Get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
func load() error {
	u, err := url.Parse(*address)
	if err != nil || u.Hostname() != "127.0.0.1" || u.Scheme != "http" {
		return fmt.Errorf("load URL must be loopback")
	}
	client := &http.Client{Timeout: 15 * time.Second}
	list := "/api/v1/notes?limit=50&offset=80000"
	if *optimized {
		list = "/api/v1/notes?limit=50&before_id=20001&before_time=" + url.QueryEscape(cursor.Format(time.RFC3339Nano))
	}
	for _, scenario := range []struct {
		name, path string
		want       int
	}{{"list", list, 50}, {"get", "/api/v1/notes/1", 1}, {"batch", "/api/v1/notes/batch?ids=1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20", 20}} {
		var before, after sql.DBStats
		if err = getJSON(client, *address+"/stats", &before); err != nil {
			return err
		}
		// 10 untimed warmup requests, then 200 closed-loop requests, concurrency 8.
		for i := 0; i < 10; i++ {
			var raw json.RawMessage
			if err = getJSON(client, *address+scenario.path, &raw); err != nil {
				return err
			}
		}
		latencies := make([]float64, 200)
		failures := 0
		var mu sync.Mutex
		var wg sync.WaitGroup
		jobs := make(chan int)
		start := time.Now()
		for worker := 0; worker < 8; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for idx := range jobs {
					t0 := time.Now()
					var gotErr error
					if scenario.name == "get" {
						var n notes.Note
						gotErr = getJSON(client, *address+scenario.path, &n)
						if gotErr == nil && n.ID != 1 {
							gotErr = fmt.Errorf("wrong id")
						}
					} else {
						var out []notes.Note
						gotErr = getJSON(client, *address+scenario.path, &out)
						if gotErr == nil {
							if len(out) != scenario.want {
								gotErr = fmt.Errorf("wrong count")
							} else {
								for i, n := range out {
									expected := int64(20000 - i)
									if scenario.name == "batch" {
										expected = int64(i + 1)
									}
									if n.ID != expected {
										gotErr = fmt.Errorf("wrong row")
									}
								}
							}
						}
					}
					latencies[idx] = float64(time.Since(t0)) / float64(time.Millisecond)
					if gotErr != nil {
						mu.Lock()
						failures++
						mu.Unlock()
					}
				}
			}()
		}
		for i := range latencies {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		elapsed := time.Since(start)
		if err = getJSON(client, *address+"/stats", &after); err != nil {
			return err
		}
		sort.Float64s(latencies)
		entry := map[string]any{"scenario": scenario.name, "optimized": *optimized, "requests": 200, "concurrency": 8, "elapsed_seconds": elapsed.Seconds(), "rps": 200 / elapsed.Seconds(), "p95_ms": latencies[189], "p99_ms": latencies[197], "errors": failures, "error_rate": float64(failures) / 200, "pool_max": after.MaxOpenConnections, "pool_open": after.OpenConnections, "pool_idle": after.Idle, "pool_in_use": after.InUse, "wait_count_delta": after.WaitCount - before.WaitCount, "wait_ms_delta": float64(after.WaitDuration-before.WaitDuration) / float64(time.Millisecond)}
		json.NewEncoder(logWriter{}).Encode(entry)
		if failures > 0 {
			return fmt.Errorf("load correctness failed")
		}
	}
	return nil
}

// Standard output contains metrics only, never connection configuration.
type logWriter struct{}

func (logWriter) Write(b []byte) (int, error) { return fmt.Print(string(b)) }
