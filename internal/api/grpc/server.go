package grpc

import (
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/bentalebwael/faceit-users-service/internal/api/grpc/gen/user"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

type Server struct {
	server  *grpc.Server
	port    int
	logger  *slog.Logger
	service *user.Service
}

func NewServer(port int, service *user.Service, logger *slog.Logger, opts ...grpc.ServerOption) *Server {
	grpcServer := grpc.NewServer(opts...)

	server := &Server{
		server:  grpcServer,
		port:    port,
		logger:  logger,
		service: service,
	}

	pb.RegisterUserServiceServer(grpcServer, NewUserServer(service, logger))

	// Register reflection service for development tools
	reflection.Register(grpcServer)

	return server
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.server.GracefulStop()
}
