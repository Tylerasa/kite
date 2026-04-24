package apiutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

// BindingError converts a gin/validator binding error into a human-readable message.
func BindingError(err error) string {
	var jsonTypeErr *json.UnmarshalTypeError
	if errors.As(err, &jsonTypeErr) {
		return fmt.Sprintf("%s has an invalid value", strings.ToLower(jsonTypeErr.Field))
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return "invalid request"
	}
	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		field := strings.ToLower(fe.Field())
		switch fe.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("%s is required", field))
		case "min":
			if field == "password" {
				msgs = append(msgs, "password must be at least 8 characters")
			} else if field == "pin" {
				msgs = append(msgs, "pin must be 4–6 digits")
			} else {
				msgs = append(msgs, fmt.Sprintf("%s must be at least %s", field, fe.Param()))
			}
		case "max":
			if field == "amount" || field == "amount_in" {
				msgs = append(msgs, "amount exceeds the maximum allowed value")
			} else {
				msgs = append(msgs, fmt.Sprintf("%s is too long", field))
			}
		case "len":
			msgs = append(msgs, fmt.Sprintf("%s must be exactly %s characters", field, fe.Param()))
		case "email":
			msgs = append(msgs, "invalid email address")
		default:
			msgs = append(msgs, fmt.Sprintf("%s is invalid", field))
		}
	}
	return strings.Join(msgs, "; ")
}

func DomainErrorStatus(code string) int {
	switch code {
	case "insufficient_funds":
		return http.StatusUnprocessableEntity
	case "quote_expired", "invalid_currency", "invalid_bank_code", "missing_idempotency_key", "validation_error":
		return http.StatusBadRequest
	case "quote_already_executed", "duplicate_transaction":
		return http.StatusConflict
	case "user_not_found", "quote_not_found", "payout_not_found", "account_not_found", "transaction_not_found":
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
