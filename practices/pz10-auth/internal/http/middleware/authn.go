package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	jwtutil "github.com/Pondaaaaaa/GO/practices/pz10-auth/internal/platform/jwt"
)

type claimsKey struct{}

func ClaimsFrom(ctx context.Context) (*jwtutil.Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*jwtutil.Claims)
	return c, ok
}
func errorJSON(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func AuthN(manager *jwtutil.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fields := strings.Fields(r.Header.Get("Authorization"))
			if len(fields) != 2 || fields[0] != "Bearer" {
				errorJSON(w, 401, "unauthorized")
				return
			}
			claims, err := manager.Parse(fields[1], jwtutil.Access)
			if err != nil {
				errorJSON(w, 401, "unauthorized")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}
}
