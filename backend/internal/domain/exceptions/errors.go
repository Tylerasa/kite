package exceptions

import "fmt"

// DomainError is a typed business error with a machine-readable code.
type DomainError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithDetails returns a copy of the error with extra context attached.
func (e *DomainError) WithDetails(details map[string]interface{}) *DomainError {
	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}

// Sentinel domain errors — use errors.Is() to compare.
var (
	ErrInsufficientFunds = &DomainError{
		Code:    "insufficient_funds",
		Message: "Insufficient balance to complete this operation.",
	}
	ErrQuoteExpired = &DomainError{
		Code:    "quote_expired",
		Message: "The FX quote has expired. Please request a new quote.",
	}
	ErrQuoteAlreadyExecuted = &DomainError{
		Code:    "quote_already_executed",
		Message: "This FX quote has already been executed.",
	}
	ErrQuoteNotFound = &DomainError{
		Code:    "quote_not_found",
		Message: "FX quote not found.",
	}
	ErrDuplicateDeposit = &DomainError{
		Code:    "duplicate_deposit",
		Message: "A deposit with this idempotency key has already been processed.",
	}
	ErrInvalidCurrency = &DomainError{
		Code:    "invalid_currency",
		Message: "The specified currency is not supported.",
	}
	ErrAccountNotFound = &DomainError{
		Code:    "account_not_found",
		Message: "Ledger account not found.",
	}
	ErrUserNotFound = &DomainError{
		Code:    "user_not_found",
		Message: "User not found.",
	}
	ErrUserAlreadyExists = &DomainError{
		Code:    "user_already_exists",
		Message: "A user with this email already exists.",
	}
	ErrInvalidCredentials = &DomainError{
		Code:    "invalid_credentials",
		Message: "Invalid email or password.",
	}
	ErrPayoutNotFound = &DomainError{
		Code:    "payout_not_found",
		Message: "Payout not found.",
	}
	ErrComplianceHold = &DomainError{
		Code:    "compliance_hold",
		Message: "This payout has been flagged for compliance review.",
	}
	ErrInternal = &DomainError{
		Code:    "internal_error",
		Message: "An unexpected error occurred.",
	}
)
