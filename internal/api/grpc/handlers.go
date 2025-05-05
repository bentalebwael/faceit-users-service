package grpc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	userpb "github.com/bentalebwael/faceit-users-service/internal/api/grpc/gen/user"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
	"github.com/bentalebwael/faceit-users-service/internal/platform/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	service *user.Service
	logger  *slog.Logger
	tracer  trace.Tracer
}

// NewUserServer creates a new UserServer
func NewUserServer(service *user.Service, logger *slog.Logger) *UserServer {
	return &UserServer{
		service: service,
		logger:  logger,
		tracer:  tracer.GetTracer(),
	}
}

// CreateUser handles the CreateUser gRPC request
func (s *UserServer) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.User, error) {
	ctx, span := s.tracer.Start(ctx, "grpc.CreateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("user.nickname", req.Nickname),
		attribute.String("user.country", req.Country),
	)
	reqUser := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Password:  req.Password,
		Email:     req.Email,
		Country:   req.Country,
	}

	newUser, err := s.service.CreateUser(ctx, reqUser)
	if err != nil {
		tracer.AddError(span, err)
		return nil, s.handleServiceError(ctx, err, "CreateUser")
	}
	return toProtoUser(newUser), nil
}

// GetUser handles the GetUser gRPC request
func (s *UserServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.User, error) {
	ctx, span := s.tracer.Start(ctx, "grpc.GetUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))
	userID, err := uuid.Parse(req.Id)
	if err != nil {
		s.logger.Warn("invalid user ID format in gRPC request", "id", req.Id, "error", err)
		tracer.AddError(span, err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID format: %v", err)
	}

	foundUser, err := s.service.GetUser(ctx, userID)
	if err != nil {
		tracer.AddError(span, err)
		return nil, s.handleServiceError(ctx, err, "GetUser")
	}
	return toProtoUser(foundUser), nil
}

// UpdateUser handles the UpdateUser gRPC request
func (s *UserServer) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.User, error) {
	ctx, span := s.tracer.Start(ctx, "grpc.UpdateUser")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", req.Id),
		attribute.String("user.email", req.Email),
		attribute.String("user.nickname", req.Nickname),
	)
	userID, err := uuid.Parse(req.Id)
	if err != nil {
		s.logger.Warn("invalid user ID format in gRPC request", "id", req.Id, "error", err)
		tracer.AddError(span, err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID format: %v", err)
	}

	updateUserReq := &user.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Country:   req.Country,
	}

	updatedUser, err := s.service.UpdateUser(ctx, userID, updateUserReq)
	if err != nil {
		tracer.AddError(span, err)
		return nil, s.handleServiceError(ctx, err, "UpdateUser")
	}
	return toProtoUser(updatedUser), nil
}

// DeleteUser handles the DeleteUser gRPC request
func (s *UserServer) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*emptypb.Empty, error) {
	ctx, span := s.tracer.Start(ctx, "grpc.DeleteUser")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", req.Id))
	userID, err := uuid.Parse(req.Id)
	if err != nil {
		s.logger.Warn("invalid user ID format in gRPC request", "id", req.Id, "error", err)
		tracer.AddError(span, err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID format: %v", err)
	}

	err = s.service.DeleteUser(ctx, userID)
	if err != nil {
		tracer.AddError(span, err)
		return nil, s.handleServiceError(ctx, err, "DeleteUser")
	}
	return &emptypb.Empty{}, nil
}

// ListUsers handles the ListUsers gRPC request
func (s *UserServer) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	ctx, span := s.tracer.Start(ctx, "grpc.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("page", int64(req.Page)),
		attribute.Int64("limit", int64(req.Limit)),
		attribute.Bool("order_desc", req.OrderDesc),
		attribute.String("order_by", req.OrderBy),
	)
	page := 1
	if req.Page > 0 {
		page = int(req.Page)
	}

	limit := 10
	if req.Limit > 0 {
		limit = int(req.Limit)
	}

	// Prepare domain ListParams
	params := user.ListParams{
		Limit:     limit,
		Offset:    (page - 1) * limit,
		OrderBy:   req.OrderBy,
		OrderDesc: req.OrderDesc,
		Filters:   make([]user.Filter, 0),
	}

	// Convert proto filters to domain filters
	for _, filter := range req.Filters {
		if filter.Value != "" {
			params.Filters = append(params.Filters, user.Filter{
				Field: filter.Field,
				Value: filter.Value,
			})
		}
	}

	users, hasMore, totalCount, err := s.service.ListUsers(ctx, params)
	if err != nil {
		tracer.AddError(span, err)
		return nil, s.handleServiceError(ctx, err, "ListUsers")
	}

	protoUsers := make([]*userpb.User, len(users))
	for i, u := range users {
		protoUsers[i] = toProtoUser(&u)
	}

	return &userpb.ListUsersResponse{
		Users:      protoUsers,
		HasMore:    hasMore,
		TotalCount: totalCount,
	}, nil
}

// handleServiceError maps domain errors to gRPC status codes
func (s *UserServer) handleServiceError(ctx context.Context, err error, methodName string) error {
	s.logger.Error("gRPC service error", "method", methodName, "error", err)

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		tracer.AddError(span, err)
	}

	switch {
	case errors.Is(err, user.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, user.ErrEmailTaken), errors.Is(err, user.ErrNicknameTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, user.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "An internal server error occurred")
	}
}

// toProtoUser converts a domain user to a gRPC user message
func toProtoUser(u *user.User) *userpb.User {
	return &userpb.User{
		Id:        u.ID.String(),
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Nickname:  u.Nickname,
		Email:     u.Email,
		Country:   u.Country,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}
