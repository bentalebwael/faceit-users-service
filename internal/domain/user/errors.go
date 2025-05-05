package user

import (
	"errors"
	"fmt"
)

// Common error types for the user domain
var (
	ErrNotFound      = fmt.Errorf("user not found")
	ErrAlreadyExists = fmt.Errorf("user already exists")
	ErrInvalidInput  = fmt.Errorf("invalid input")
	ErrEmailTaken    = fmt.Errorf("email is already taken")
	ErrNicknameTaken = fmt.Errorf("nickname is already taken")
	ErrValidation    = fmt.Errorf("validation error")
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists) || errors.Is(err, ErrEmailTaken) || errors.Is(err, ErrNicknameTaken)
}

func IsValidationError(err error) bool {
	var validationError *ValidationError
	ok := errors.As(err, &validationError)
	return ok || errors.Is(err, ErrValidation)
}
