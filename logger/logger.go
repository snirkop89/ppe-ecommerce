package logger

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

func NewLogger(service string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stderr, nil)
	l := slog.New(h.WithAttrs([]slog.Attr{{Key: "service", Value: slog.StringValue(service)}}))
	return l
}

func LoggingMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attributes := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("user-agent", r.UserAgent()),
				slog.String("path", r.URL.Path),
			}

			start := time.Now()
			next.ServeHTTP(w, r)

			attributes = append(attributes, slog.String("latency", time.Since(start).String()))

			logger.WithGroup("http").LogAttrs(r.Context(), slog.LevelInfo, "Handled request", attributes...)
		})
	}
}
