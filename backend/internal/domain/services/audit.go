package services

import "context"

// contextKey is an unexported type for context keys in the domain layer.
type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestIDFromCtx extracts the request ID threaded by the HTTP middleware.
func RequestIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(RequestIDKey).(string)
	return v
}
