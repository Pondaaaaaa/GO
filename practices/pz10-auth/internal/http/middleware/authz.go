package middleware

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func AuthZRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFrom(r.Context())
			if !ok {
				errorJSON(w, 401, "unauthorized")
				return
			}
			if !allowed[claims.Role] {
				errorJSON(w, 403, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func OwnProfileOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFrom(r.Context())
		if !ok {
			errorJSON(w, 401, "unauthorized")
			return
		}
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil || id <= 0 {
			errorJSON(w, 400, "invalid_id")
			return
		}
		if claims.Role != "admin" && claims.Subject != strconv.FormatInt(id, 10) {
			errorJSON(w, 403, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
