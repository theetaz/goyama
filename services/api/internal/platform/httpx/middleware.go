package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// JSON writes a value as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// Problem renders an RFC 7807 problem+json payload.
func Problem(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	payload := map[string]any{
		"type":       "https://cropdoc.lk/errors/" + title,
		"title":      title,
		"status":     status,
		"detail":     detail,
		"instance":   r.URL.Path,
		"request_id": RequestIDFromContext(r),
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// RequestIDMiddleware generates a ULID-like ID per request and stores it in the
// response header + context. We use uuid.NewV7 for time-ordered IDs.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.NewV7()
		var rid string
		if err == nil {
			rid = id.String()
		} else {
			rid = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", rid)
		ctx := WithRequestID(r.Context(), rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AccessLog logs one structured line per request.
func AccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", w.Header().Get("X-Request-ID")),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}
