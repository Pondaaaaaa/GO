package notes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func response(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, err error) {
	status, msg := 500, "database operation failed"
	if errors.Is(err, ErrInvalid) {
		status, msg = 400, "invalid request"
	}
	if errors.Is(err, sql.ErrNoRows) {
		status, msg = 404, "note not found"
	}
	response(w, status, map[string]string{"error": msg})
}
func decode(r *http.Request, v any) error {
	typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typ != "application/json" {
		return ErrInvalid
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrInvalid
	}
	return nil
}
func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		return 0, ErrInvalid
	}
	return id, nil
}
func Router(repo *Repo) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if repo.DB.PingContext(r.Context()) != nil {
			fail(w, errors.New("ping"))
			return
		}
		response(w, 200, map[string]bool{"ok": true})
	})
	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) { response(w, 200, repo.DB.Stats()) })
	r.Post("/api/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		var in Input
		if decode(r, &in) != nil {
			fail(w, ErrInvalid)
			return
		}
		notes, err := repo.CreateMany(r.Context(), []Input{in})
		if err != nil {
			fail(w, err)
			return
		}
		w.Header().Set("Location", "/api/v1/notes/"+strconv.FormatInt(notes[0].ID, 10))
		response(w, 201, notes[0])
	})
	r.Post("/api/v1/notes/many", func(w http.ResponseWriter, r *http.Request) {
		var in []Input
		if decode(r, &in) != nil {
			fail(w, ErrInvalid)
			return
		}
		out, err := repo.CreateMany(r.Context(), in)
		if err != nil {
			fail(w, err)
			return
		}
		response(w, 201, out)
	})
	r.Get("/api/v1/notes/batch", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Query().Get("ids"), ",")
		if len(parts) > 20 {
			fail(w, ErrInvalid)
			return
		}
		ids := make([]int64, 0, len(parts))
		seen := map[int64]bool{}
		for _, s := range parts {
			id, err := strconv.ParseInt(s, 10, 64)
			if err != nil || id < 1 || seen[id] {
				fail(w, ErrInvalid)
				return
			}
			ids = append(ids, id)
			seen[id] = true
		}
		out, err := repo.Batch(r.Context(), ids)
		if err != nil {
			fail(w, err)
			return
		}
		response(w, 200, out)
	})
	r.Get("/api/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, offset := 50, 0
		var err error
		if q.Has("limit") {
			limit, err = strconv.Atoi(q.Get("limit"))
			if err != nil {
				fail(w, ErrInvalid)
				return
			}
		}
		if q.Has("offset") {
			offset, err = strconv.Atoi(q.Get("offset"))
			if err != nil {
				fail(w, ErrInvalid)
				return
			}
		}
		if limit < 1 || limit > 100 || offset < 0 || offset > 100000 {
			fail(w, ErrInvalid)
			return
		}
		var ct *time.Time
		var cid int64
		if q.Has("before_time") || q.Has("before_id") {
			t, e := time.Parse(time.RFC3339Nano, q.Get("before_time"))
			cid, err = strconv.ParseInt(q.Get("before_id"), 10, 64)
			if e != nil || err != nil || cid < 1 || offset != 0 || !repo.Optimized {
				fail(w, ErrInvalid)
				return
			}
			ct = &t
		}
		out, err := repo.List(r.Context(), limit, offset, ct, cid)
		if err != nil {
			fail(w, err)
			return
		}
		response(w, 200, out)
	})
	r.Get("/api/v1/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err != nil {
			fail(w, err)
			return
		}
		n, err := repo.Get(r.Context(), id)
		if err != nil {
			fail(w, err)
			return
		}
		response(w, 200, n)
	})
	r.Patch("/api/v1/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err != nil {
			fail(w, err)
			return
		}
		var p Patch
		if decode(r, &p) != nil {
			fail(w, ErrInvalid)
			return
		}
		n, err := repo.Update(r.Context(), id, p)
		if err != nil {
			fail(w, err)
			return
		}
		response(w, 200, n)
	})
	r.Delete("/api/v1/notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r)
		if err == nil {
			err = repo.Delete(r.Context(), id)
		}
		if err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(204)
	})
	return r
}
