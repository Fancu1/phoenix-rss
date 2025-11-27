package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Fancu1/phoenix-rss/internal/config"
	"github.com/Fancu1/phoenix-rss/internal/user-service/core"
	"github.com/Fancu1/phoenix-rss/internal/user-service/handler"
	userRepo "github.com/Fancu1/phoenix-rss/internal/user-service/repository"
	"github.com/Fancu1/phoenix-rss/pkg/logger"
	userpb "github.com/Fancu1/phoenix-rss/protos/gen/go/user"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.New(slog.LevelDebug)

	// initialize database connection
	db := userRepo.InitDB(&cfg.Database)

	// initialize user repository and service
	userRepository := userRepo.NewUserRepository(db)
	userSvc := core.NewUserService(userRepository, cfg.Auth.JWTSecret)

	// create gRPC handler
	grpcHandler := handler.NewUserServiceHandler(userSvc)

	// create gRPC server
	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, grpcHandler)

	// register gRPC health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// start listening on the specified port
	port := "50051" // default port for user service
	if userServicePort := os.Getenv("USER_SERVICE_PORT"); userServicePort != "" {
		port = userServicePort
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("failed to listen", "port", port, "error", err)
		os.Exit(1)
	}

	logger.Info("User Service starting", "port", port)
	fmt.Printf("User Service listening on port %s\n", port)

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("failed to serve gRPC server", "error", err)
		os.Exit(1)
	}
}
