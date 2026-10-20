package httperror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestDescribeNormalizesHTTPStatusContract(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		status    int
		code      string
		kind      Kind
		retryable bool
		message   string
	}{
		{
			name:    "validation",
			err:     fiber.NewError(http.StatusUnprocessableEntity, "bad input"),
			status:  422,
			code:    "validation_error",
			kind:    KindValidation,
			message: "bad input",
		},
		{
			name:   "authentication",
			err:    fiber.ErrUnauthorized,
			status: 401,
			code:   "authentication_required",
			kind:   KindAuthentication,
		},
		{
			name:   "authorization",
			err:    fiber.ErrForbidden,
			status: 403,
			code:   "forbidden",
			kind:   KindAuthorization,
		},
		{
			name:   "not found",
			err:    fiber.ErrNotFound,
			status: 404,
			code:   "not_found",
			kind:   KindNotFound,
		},
		{
			name:   "conflict",
			err:    fiber.ErrConflict,
			status: 409,
			code:   "conflict",
			kind:   KindConflict,
		},
		{
			name: "dependency",
			err: New(
				KindDependency,
				"database_failure",
				"database is unavailable",
				errors.New("secret database details"),
			),
			status:    424,
			code:      "database_failure",
			kind:      KindDependency,
			retryable: true,
			message:   "database is unavailable",
		},
		{
			name:      "unavailable",
			err:       Unavailable("live_unavailable", "live race is not configured", nil),
			status:    503,
			code:      "live_unavailable",
			kind:      KindUnavailable,
			retryable: true,
			message:   "live race is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := Describe(tt.err)
			if problem.Status != tt.status || problem.Code != tt.code || problem.Kind != tt.kind ||
				problem.Retryable != tt.retryable {
				t.Fatalf("unexpected problem: %#v", problem)
			}
			if tt.message != "" && problem.Message != tt.message {
				t.Fatalf("message=%q, want %q", problem.Message, tt.message)
			}
		})
	}
}

func TestDescribeDoesNotExposeInternalCause(t *testing.T) {
	problem := Describe(errors.New("postgres password and query details"))
	if problem.Status != http.StatusInternalServerError || problem.Code != "internal_error" {
		t.Fatalf("unexpected internal problem: %#v", problem)
	}
	if problem.Message != "Something went wrong while processing your request." {
		t.Fatalf("internal cause leaked in message: %q", problem.Message)
	}
}

func TestNewUsesCanonicalStatusForEachKind(t *testing.T) {
	if got := Describe(Validation("", "", nil)); got.Status != http.StatusUnprocessableEntity {
		t.Fatalf("validation status=%d", got.Status)
	}
	if got := Describe(Conflict("", "", nil)); got.Status != http.StatusConflict {
		t.Fatalf("conflict status=%d", got.Status)
	}
	if got := Describe(Dependency("", "", nil)); got.Status != http.StatusFailedDependency {
		t.Fatalf("dependency status=%d", got.Status)
	}
}
