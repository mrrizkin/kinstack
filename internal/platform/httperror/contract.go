// Package httperror defines the public HTTP error contract shared by all
// feature handlers. Feature code may keep domain-specific errors internally,
// but the HTTP boundary exposes only this stable, non-sensitive shape.
package httperror

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// Kind identifies the client-actionable class of an HTTP failure.
type Kind string

const (
	KindValidation     Kind = "validation"
	KindAuthentication Kind = "authentication"
	KindAuthorization  Kind = "authorization"
	KindNotFound       Kind = "not_found"
	KindConflict       Kind = "conflict"
	KindDependency     Kind = "dependency"
	KindUnavailable    Kind = "unavailable"
	KindRateLimit      Kind = "rate_limit"
	KindGone           Kind = "gone"
	KindInternal       Kind = "internal"
)

// Problem is the canonical error payload sent to JSON clients and Inertia
// error pages. Message is always safe for a client; causes are logged by the
// application boundary and are never serialized.
type Problem struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Kind      Kind   `json:"kind"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// Error is a typed error for handlers that need to identify a dependency or
// unavailable feature without losing the original cause in logs.
type Error struct {
	problem Problem
	cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.problem.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// StatusCode lets outer middleware record the correct outcome before Fiber's
// final error handler writes the response.
func (e *Error) StatusCode() int {
	if e == nil {
		return 0
	}
	return e.problem.Status
}

// New creates a typed HTTP error. Status is constrained to the standard
// contract so callers cannot accidentally expose an unsupported category.
func New(kind Kind, code, message string, cause error) error {
	if !isKnownKind(kind) {
		kind = KindInternal
	}
	status := statusForKind(kind)
	if code == "" {
		code = codeForKind(kind)
	}
	if message == "" {
		message = messageForKind(kind)
	}
	return &Error{
		problem: Problem{
			Status:    status,
			Code:      code,
			Kind:      kind,
			Message:   message,
			Retryable: kind == KindDependency || kind == KindUnavailable || kind == KindRateLimit,
		},
		cause: cause,
	}
}

func Validation(code, message string, cause error) error {
	return New(KindValidation, code, message, cause)
}

func Authentication(code, message string, cause error) error {
	return New(KindAuthentication, code, message, cause)
}

func Authorization(code, message string, cause error) error {
	return New(KindAuthorization, code, message, cause)
}

func NotFound(code, message string, cause error) error {
	return New(KindNotFound, code, message, cause)
}

func Conflict(code, message string, cause error) error {
	return New(KindConflict, code, message, cause)
}

func Dependency(code, message string, cause error) error {
	return New(KindDependency, code, message, cause)
}

func Unavailable(code, message string, cause error) error {
	return New(KindUnavailable, code, message, cause)
}

// Describe normalizes both typed contract errors and legacy Fiber errors. This
// lets existing feature handlers adopt the contract incrementally without
// changing their domain error types.
func Describe(err error) Problem {
	if err == nil {
		return Problem{}
	}
	var contractErr *Error
	if errors.As(err, &contractErr) {
		return contractErr.problem
	}

	status := http.StatusInternalServerError
	message := ""
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		status = fiberErr.Code
		message = fiberErr.Message
	}
	problem := problemForStatus(status)
	if status < http.StatusInternalServerError && message != "" {
		problem.Message = message
	}
	return problem
}

func problemForStatus(status int) Problem {
	kind := kindForStatus(status)
	return Problem{
		Status:    status,
		Code:      codeForKind(kind),
		Kind:      kind,
		Message:   messageForKind(kind),
		Retryable: kind == KindDependency || kind == KindUnavailable || kind == KindRateLimit,
	}
}

func kindForStatus(status int) Kind {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return KindValidation
	case http.StatusUnauthorized:
		return KindAuthentication
	case http.StatusForbidden:
		return KindAuthorization
	case http.StatusNotFound:
		return KindNotFound
	case http.StatusConflict:
		return KindConflict
	case http.StatusFailedDependency:
		return KindDependency
	case http.StatusServiceUnavailable, http.StatusNotImplemented:
		return KindUnavailable
	case http.StatusTooManyRequests:
		return KindRateLimit
	case http.StatusGone:
		return KindGone
	default:
		return KindInternal
	}
}

func isKnownKind(kind Kind) bool {
	switch kind {
	case KindValidation, KindAuthentication, KindAuthorization, KindNotFound,
		KindConflict, KindDependency, KindUnavailable, KindRateLimit, KindGone, KindInternal:
		return true
	default:
		return false
	}
}

func statusForKind(kind Kind) int {
	switch kind {
	case KindValidation:
		return http.StatusUnprocessableEntity
	case KindAuthentication:
		return http.StatusUnauthorized
	case KindAuthorization:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindDependency:
		return http.StatusFailedDependency
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindRateLimit:
		return http.StatusTooManyRequests
	case KindGone:
		return http.StatusGone
	default:
		return http.StatusInternalServerError
	}
}

func codeForKind(kind Kind) string {
	switch kind {
	case KindValidation:
		return "validation_error"
	case KindAuthentication:
		return "authentication_required"
	case KindAuthorization:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindDependency:
		return "dependency_failure"
	case KindUnavailable:
		return "feature_unavailable"
	case KindRateLimit:
		return "rate_limited"
	case KindGone:
		return "gone"
	default:
		return "internal_error"
	}
}

func messageForKind(kind Kind) string {
	switch kind {
	case KindValidation:
		return "The request could not be validated."
	case KindAuthentication:
		return "Authentication is required."
	case KindAuthorization:
		return "You are not allowed to perform this action."
	case KindNotFound:
		return "The requested resource was not found."
	case KindConflict:
		return "The resource changed or the request was already processed."
	case KindDependency:
		return "A required dependency failed. Please try again."
	case KindUnavailable:
		return "This feature is temporarily unavailable."
	case KindRateLimit:
		return "Too many requests. Please try again later."
	case KindGone:
		return "This resource is no longer available."
	default:
		return "Something went wrong while processing your request."
	}
}
