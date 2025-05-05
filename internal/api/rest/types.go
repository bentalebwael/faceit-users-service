package rest

import "github.com/bentalebwael/faceit-users-service/internal/domain/user"

// AddUserRequest represents the request to create a new user
type AddUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Nickname  string `json:"nickname" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Country   string `json:"country" binding:"required,len=2"`
}

// UpdateUserRequest represents the request to update a user
// Fields are optional to allow partial updates.
type UpdateUserRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	Email     string `json:"email,omitempty" binding:"omitempty,email"`   // validate if present
	Country   string `json:"country,omitempty" binding:"omitempty,len=2"` // validate if present
}

// ListUsersResponse represents the paginated response for listing users using offset pagination
type ListUsersResponse struct {
	Users      []user.User `json:"users"`
	HasMore    bool        `json:"has_more"`
	TotalCount int64       `json:"total_count"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}
