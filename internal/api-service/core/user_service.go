package core

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Fancu1/phoenix-rss/internal/user-service/models"
	userpb "github.com/Fancu1/phoenix-rss/protos/gen/go/user"
)

// UserServiceInterface define the contract for user service operations
type UserServiceInterface interface {
	Register(username, password string) (*models.User, error)
	Login(username, password string) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetUserFromToken(tokenString string) (*models.User, error)
}

// UserServiceClient implement UserServiceInterface using gRPC
type UserServiceClient struct {
	client userpb.UserServiceClient
	conn   *grpc.ClientConn
}

// NewUserServiceClient create a new gRPC client for the user service
func NewUserServiceClient(address string) (*UserServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service at %s: %w", address, err)
	}

	client := userpb.NewUserServiceClient(conn)

	return &UserServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close close the gRPC connection
func (c *UserServiceClient) Close() error {
	return c.conn.Close()
}

func (c *UserServiceClient) Register(username, password string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &userpb.RegisterRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.client.Register(ctx, req)
	if err != nil {
		return nil, MapGRPCError(err)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("user service returned nil user")
	}

	return &models.User{
		ID:       uint(resp.User.Id),
		Username: resp.User.Username,
	}, nil
}

func (c *UserServiceClient) Login(username, password string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &userpb.LoginRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.client.Login(ctx, req)
	if err != nil {
		return "", MapGRPCError(err)
	}

	return resp.Token, nil
}

func (c *UserServiceClient) ValidateToken(tokenString string) (*jwt.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &userpb.ValidateTokenRequest{
		Token: tokenString,
	}

	resp, err := c.client.ValidateToken(ctx, req)
	if err != nil {
		return nil, MapGRPCError(err)
	}

	if !resp.Valid {
		return nil, fmt.Errorf("token validation failed: %s", resp.Error)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("user service returned nil user for valid token")
	}

	// create a mock JWT token since we only need the validation result
	// the actual token parsing and validation is done by the user service
	token := &jwt.Token{
		Valid: resp.Valid,
		Claims: jwt.MapClaims{
			"user_id":  float64(resp.User.Id),
			"username": resp.User.Username,
		},
	}

	return token, nil
}

func (c *UserServiceClient) GetUserFromToken(tokenString string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &userpb.GetUserFromTokenRequest{
		Token: tokenString,
	}

	resp, err := c.client.GetUserFromToken(ctx, req)
	if err != nil {
		return nil, MapGRPCError(err)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("user service returned nil user")
	}

	return &models.User{
		ID:       uint(resp.User.Id),
		Username: resp.User.Username,
	}, nil
}
