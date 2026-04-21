package apiutil

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kite/internal/domain/exceptions"
)

// ErrorResponse is the canonical error shape for all API responses.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// RespondError maps a domain error to a structured HTTP response.
func RespondError(c *gin.Context, logger *slog.Logger, err error) {
	requestID, _ := c.Get("request_id")

	var domainErr *exceptions.DomainError
	if errors.As(err, &domainErr) {
		status := DomainErrorStatus(domainErr.Code)

		if status >= 500 {
			logger.Error("internal error", "request_id", requestID, "error", err)
		} else {
			logger.Warn("domain error", "request_id", requestID, "code", domainErr.Code)
		}

		c.JSON(status, ErrorResponse{Error: ErrorBody{
			Code:    domainErr.Code,
			Message: domainErr.Message,
			Details: domainErr.Details,
		}})
		return
	}

	logger.Error("unexpected error", "request_id", requestID, "error", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorBody{
		Code:    "internal_error",
		Message: "An unexpected error occurred.",
	}})
}

func DomainErrorStatus(code string) int {
	switch code {
	case "insufficient_funds":
		return http.StatusUnprocessableEntity
	case "quote_expired", "invalid_currency", "missing_idempotency_key", "validation_error":
		return http.StatusBadRequest
	case "quote_already_executed":
		return http.StatusConflict
	case "user_not_found", "quote_not_found", "payout_not_found", "account_not_found":
		return http.StatusNotFound
	case "user_already_exists":
		return http.StatusConflict
	case "invalid_credentials", "unauthorized":
		return http.StatusUnauthorized
	case "compliance_hold":
		return http.StatusAccepted
	default:
		return http.StatusInternalServerError
	}
}
