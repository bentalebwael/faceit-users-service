package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/bentalebwael/faceit-users-service/internal/api"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

type Handler struct {
	service       *user.Service
	healthChecker *api.HealthChecker
	logger        *slog.Logger
}

func NewHandler(service *user.Service, healthChecker *api.HealthChecker, logger *slog.Logger) *Handler {
	return &Handler{
		service:       service,
		healthChecker: healthChecker,
		logger:        logger,
	}
}

// Health handles GET /healthz requests
func (h *Handler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	health := h.healthChecker.Check(ctx)
	if health.Status == api.Healthy {
		c.JSON(http.StatusOK, health)
	} else {
		c.JSON(http.StatusServiceUnavailable, health)
	}
}

// AddUser handles POST /users requests
func (h *Handler) AddUser(c *gin.Context) {
	ctx := c.Request.Context()
	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "bad_request", Message: err.Error()})
		return
	}

	reqUser := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Password:  req.Password,
		Email:     req.Email,
		Country:   req.Country,
	}

	newUser, err := h.service.CreateUser(ctx, reqUser)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newUser)
}

// GetUser handles GET /users/:id requests
func (h *Handler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("invalid user ID format", "id", idStr, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "bad_request", Message: "Invalid user ID format"})
		return
	}

	user, err := h.service.GetUser(ctx, userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles PUT /users/:id requests
func (h *Handler) UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("invalid user ID format", "id", idStr, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "bad_request", Message: "Invalid user ID format"})
		return
	}

	var req UpdateUserRequest
	// Use ShouldBindJSON which respects binding tags (like omitempty for validation)
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("failed to bind update request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "bad_request", Message: err.Error()})
		return
	}

	updateUserReq := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Country:   req.Country,
	}

	updatedUser, err := h.service.UpdateUser(ctx, userID, updateUserReq)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser handles DELETE /users/:id requests
func (h *Handler) DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("invalid user ID format", "id", idStr, "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Code: "bad_request", Message: "Invalid user ID format"})
		return
	}

	err = h.service.DeleteUser(ctx, userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUsers handles GET /users requests
func (h *Handler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1 // Default value
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 10 // Default value
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	params := user.ListParams{
		Limit:     limit,
		Offset:    (page - 1) * limit,
		OrderBy:   "created_at", // Default value
		OrderDesc: true,         // Default value
		Filters:   make([]user.Filter, 0),
	}

	if orderBy := c.Query("order_by"); orderBy != "" {
		params.OrderBy = orderBy
	}

	if orderDescStr := c.Query("order_desc"); orderDescStr != "" {
		if boolValue, err := strconv.ParseBool(orderDescStr); err == nil {
			params.OrderDesc = boolValue
		}
	}

	// Parse filters from query parameters
	queryParams := c.Request.URL.Query()
	for key, values := range queryParams {
		if len(values) > 0 && values[0] != "" {
			value := values[0]
			params.Filters = append(params.Filters, user.Filter{
				Field: key,
				Value: value,
			})
		}
	}

	users, hasMore, totalCount, err := h.service.ListUsers(ctx, params)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	resp := ListUsersResponse{
		Users:      users,
		HasMore:    hasMore,
		TotalCount: totalCount,
	}

	c.JSON(http.StatusOK, resp)
}

// handleServiceError maps domain errors to HTTP status codes
func (h *Handler) handleServiceError(c *gin.Context, err error) {
	h.logger.Error("service error", "error", err)

	var code string
	var status int
	var message string = err.Error() // Default message

	switch {
	case errors.Is(err, user.ErrNotFound):
		code = "not_found"
		status = http.StatusNotFound
	case errors.Is(err, user.ErrEmailTaken), errors.Is(err, user.ErrNicknameTaken):
		code = "conflict"
		status = http.StatusConflict
	case errors.Is(err, user.ErrValidation):
		code = "bad_request"
		status = http.StatusBadRequest
	default:
		// Fallback for unexpected errors
		code = "internal_error"
		status = http.StatusInternalServerError
		message = "An internal error occurred"
	}

	c.JSON(status, ErrorResponse{Code: code, Message: message})
}
