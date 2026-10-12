package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

const RequestIDHeader = "X-Request-Id"

func Info(message string, args ...any) {
	slog.Info(message, args...)
}

func Error(message string, args ...any) {
	slog.Error(message, args...)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now().UTC()
		next.ServeHTTP(w, r)
		Info("HTTP request",
			"time", started.Format(time.RFC3339),
			"remote_addr", r.RemoteAddr,
			"method", r.Method,
			"path", r.URL.Path,
		)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			var randomBytes [8]byte
			if _, err := rand.Read(randomBytes[:]); err != nil {
				Error("generate request ID", "error", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			requestID = hex.EncodeToString(randomBytes[:])
		}
		w.Header().Set(RequestIDHeader, requestID)
		next.ServeHTTP(w, r)
	})
}
