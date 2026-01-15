package errors

import (
	"fmt"
)

// AppError represents an application error with a user-safe message and optional internal details.
type AppError struct {
	Code       string // Machine-readable error code
	Message    string // User-safe error message
	HTTPStatus int    // HTTP status code to return
	Internal   error  // Internal error details (not exposed to client)
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// UserMessage returns the user-safe message.
func (e *AppError) UserMessage() string {
	return e.Message
}

// Common error codes
const (
	ErrCodeValidationFailed   = "VALIDATION_FAILED"
	ErrCodeInvalidRequest     = "INVALID_REQUEST"
	ErrCodeRecaptchaFailed    = "RECAPTCHA_FAILED"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
)

// Predefined errors
func NewValidationError(message string, internal error) *AppError {
	return &AppError{
		Code:       ErrCodeValidationFailed,
		Message:    message,
		HTTPStatus: 400,
		Internal:   internal,
	}
}

func NewRecaptchaError(message string, internal error) *AppError {
	return &AppError{
		Code:       ErrCodeRecaptchaFailed,
		Message:    message,
		HTTPStatus: 502,
		Internal:   internal,
	}
}

func NewInternalError(message string, internal error) *AppError {
	return &AppError{
		Code:       ErrCodeInternalError,
		Message:    message,
		HTTPStatus: 500,
		Internal:   internal,
	}
}
