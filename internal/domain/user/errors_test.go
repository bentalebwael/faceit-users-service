package user

import (
	"errors"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "not found error",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "other error",
			err:  ErrAlreadyExists,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "already exists error",
			err:  ErrAlreadyExists,
			want: true,
		},
		{
			name: "email taken error",
			err:  ErrEmailTaken,
			want: true,
		},
		{
			name: "nickname taken error",
			err:  ErrNicknameTaken,
			want: true,
		},
		{
			name: "other error",
			err:  ErrNotFound,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAlreadyExists(tt.err); got != tt.want {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "validation error type",
			err:  NewValidationError("field", "message"),
			want: true,
		},
		{
			name: "generic validation error",
			err:  ErrValidation,
			want: true,
		},
		{
			name: "other error",
			err:  ErrNotFound,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidationError(tt.err); got != tt.want {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
		want    string
	}{
		{
			name:    "standard validation error",
			field:   "email",
			message: "invalid format",
			want:    "validation error: email - invalid format",
		},
		{
			name:    "empty field",
			field:   "",
			message: "missing field",
			want:    "validation error:  - missing field",
		},
		{
			name:    "empty message",
			field:   "username",
			message: "",
			want:    "validation error: username - ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ValidationError{
				Field:   tt.field,
				Message: tt.message,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
	}{
		{
			name:    "standard error",
			field:   "email",
			message: "invalid format",
		},
		{
			name:    "empty field",
			field:   "",
			message: "missing field",
		},
		{
			name:    "empty message",
			field:   "username",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.message)
			var ve *ValidationError
			ok := errors.As(err, &ve)
			if !ok {
				t.Errorf("NewValidationError() did not return a *ValidationError")
				return
			}
			if ve.Field != tt.field {
				t.Errorf("NewValidationError().Field = %v, want %v", ve.Field, tt.field)
			}
			if ve.Message != tt.message {
				t.Errorf("NewValidationError().Message = %v, want %v", ve.Message, tt.message)
			}
		})
	}
}
