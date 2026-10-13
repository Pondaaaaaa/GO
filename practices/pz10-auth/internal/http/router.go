package router

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/core"
	"github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/http/middleware"
	jwtutil "github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/platform/jwt"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Users     core.UserRepository
	Tokens    *jwtutil.Manager
	Blacklist *core.RefreshBlacklist
}

func Build(s *Service) http.Handler {
	r := chi.NewRouter()
	r.Use(logRequest)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	r.Post("/api/v1/login", s.login)
	r.Post("/api/v1/refresh", s.refresh)
	r.Group(func(p chi.Router) {
		p.Use(middleware.AuthN(s.Tokens))
		p.Use(middleware.AuthZRoles("admin", "user"))
		p.Get("/api/v1/me", s.me)
		p.With(middleware.OwnProfileOrAdmin).Get("/api/v1/users/{id}", s.profile)
	})
	r.Group(func(a chi.Router) {
		a.Use(middleware.AuthN(s.Tokens))
		a.Use(middleware.AuthZRoles("admin"))
		a.Get("/api/v1/admin/stats", s.adminStats)
	})
	return r
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("method=%s path=%s status=%d duration=%s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}
func readJSON(r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("content type")
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 8193))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("trailing data")
	}
	return nil
}
func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil || strings.TrimSpace(in.Email) == "" || in.Password == "" {
		writeError(w, 400, "invalid_request")
		return
	}
	user, found := s.Users.ByEmail(in.Email)
	if !found || bcrypt.CompareHashAndPassword(user.Hash, []byte(in.Password)) != nil {
		writeError(w, 401, "invalid_credentials")
		return
	}
	s.issuePair(w, user)
}
func (s *Service) issuePair(w http.ResponseWriter, user core.UserRecord) {
	access, err := s.Tokens.Sign(user.ID, user.Email, user.Role, jwtutil.Access)
	if err != nil {
		writeError(w, 500, "token_error")
		return
	}
	refresh, err := s.Tokens.Sign(user.ID, user.Email, user.Role, jwtutil.Refresh)
	if err != nil {
		writeError(w, 500, "token_error")
		return
	}
	writeJSON(w, 200, map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": int(s.Tokens.AccessTTL.Seconds())})
}
func (s *Service) refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := readJSON(r, &in); err != nil || in.RefreshToken == "" {
		writeError(w, 400, "invalid_request")
		return
	}
	claims, err := s.Tokens.Parse(in.RefreshToken, jwtutil.Refresh)
	if err != nil {
		writeError(w, 401, "unauthorized")
		return
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		writeError(w, 401, "unauthorized")
		return
	}
	user, ok := s.Users.ByID(id)
	if !ok || user.Email != claims.Email {
		writeError(w, 401, "unauthorized")
		return
	}
	if !s.Blacklist.RevokeOnce(claims.ID, claims.ExpiresAt.Time) {
		writeError(w, 401, "refresh_revoked")
		return
	}
	s.issuePair(w, user)
}
func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFrom(r.Context())
	id, _ := strconv.ParseInt(claims.Subject, 10, 64)
	writeJSON(w, 200, core.User{ID: id, Email: claims.Email, Role: claims.Role})
}
func (s *Service) profile(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	user, ok := s.Users.ByID(id)
	if !ok {
		writeError(w, 404, "not_found")
		return
	}
	writeJSON(w, 200, user.User)
}
func (s *Service) adminStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"users": 2, "version": "1.0"})
}
