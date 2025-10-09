package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(wrapped, r)

		status := fmt.Sprintf("%d", wrapped.statusCode)
		slog.Info(
			"[Request Completed]",
			"status", status,
			"method", r.Method,
			"uri", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}
