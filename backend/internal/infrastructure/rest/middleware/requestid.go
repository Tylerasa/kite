package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kite/internal/domain/services"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(RequestIDKey, id)
		c.Header("X-Request-ID", id)
		// Thread into Go context so domain use cases can read it without importing middleware.
		ctx := context.WithValue(c.Request.Context(), services.RequestIDKey, id)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
