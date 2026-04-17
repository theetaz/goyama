package httpx

import (
	"context"
	"net/http"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
)

// WithRequestID stores a request ID on the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request ID from the request context, or empty.
func RequestIDFromContext(r *http.Request) string {
	if v, ok := r.Context().Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
