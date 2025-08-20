package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Fancu1/phoenix-rss/internal/ierr"
	"github.com/Fancu1/phoenix-rss/internal/user-service/core"
	userpb "github.com/Fancu1/phoenix-rss/protos/gen/go/user"
)

type UserServiceHandler struct {
	userpb.UnimplementedUserServiceServer
	userService core.UserServiceInterface
}

func NewUserServiceHandler(userService core.UserServiceInterface) *UserServiceHandler {
	return &UserServiceHandler{
		userService: userService,
	}
}

func (h *UserServiceHandler) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	// validate input
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// call the business logic
	registeredUser, err := h.userService.Register(req.Username, req.Password)
	if err != nil {
		return nil, h.handleError(err)
	}

	// convert to proto response
	return &userpb.RegisterResponse{
		User: &userpb.User{
			Id:       uint64(registeredUser.ID),
			Username: registeredUser.Username,
		},
	}, nil
}

func (h *UserServiceHandler) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	// validate input
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// call the business logic
	token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		return nil, h.handleError(err)
	}

	// get user details for response
	userFromToken, err := h.userService.GetUserFromToken(token)
	if err != nil {
		return nil, h.handleError(err)
	}

	// convert to proto response
	return &userpb.LoginResponse{
		Token: token,
		User: &userpb.User{
			Id:       uint64(userFromToken.ID),
			Username: userFromToken.Username,
		},
	}, nil
}

func (h *UserServiceHandler) ValidateToken(ctx context.Context, req *userpb.ValidateTokenRequest) (*userpb.ValidateTokenResponse, error) {
	// validate input
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// call the business logic
	token, err := h.userService.ValidateToken(req.Token)
	if err != nil {
		return &userpb.ValidateTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	// extract user from token for response
	userFromToken, err := h.userService.GetUserFromToken(req.Token)
	if err != nil {
		return &userpb.ValidateTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	// convert to proto response
	return &userpb.ValidateTokenResponse{
		Valid: token.Valid,
		Error: "",
		User: &userpb.User{
			Id:       uint64(userFromToken.ID),
			Username: userFromToken.Username,
		},
	}, nil
}

func (h *UserServiceHandler) GetUserFromToken(ctx context.Context, req *userpb.GetUserFromTokenRequest) (*userpb.GetUserFromTokenResponse, error) {
	// validate input
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// call the business logic
	userFromToken, err := h.userService.GetUserFromToken(req.Token)
	if err != nil {
		return nil, h.handleError(err)
	}

	// convert to proto response
	return &userpb.GetUserFromTokenResponse{
		User: &userpb.User{
			Id:       uint64(userFromToken.ID),
			Username: userFromToken.Username,
		},
	}, nil
}

// handleError converts internal errors to appropriate gRPC status codes
func (h *UserServiceHandler) handleError(err error) error {
	// check for specific error types
	var appErr *ierr.AppError
	if errors.As(err, &appErr) {
		switch appErr {
		case ierr.ErrUserExists:
			return status.Error(codes.AlreadyExists, appErr.Error())
		case ierr.ErrInvalidCredentials:
			return status.Error(codes.Unauthenticated, appErr.Error())
		case ierr.ErrInvalidToken:
			return status.Error(codes.Unauthenticated, appErr.Error())
		case ierr.ErrUserNotFound:
			return status.Error(codes.NotFound, appErr.Error())
		case ierr.ErrUnauthorized:
			return status.Error(codes.Unauthenticated, appErr.Error())
		case ierr.ErrDatabaseError:
			return status.Error(codes.Internal, "database error occurred")
		case ierr.ErrInternalServer:
			return status.Error(codes.Internal, "internal server error")
		default:
			// check by status code for general categories
			if appErr.HTTPStatus == http.StatusBadRequest {
				return status.Error(codes.InvalidArgument, appErr.Error())
			}
			if appErr.HTTPStatus == http.StatusUnauthorized {
				return status.Error(codes.Unauthenticated, appErr.Error())
			}
			if appErr.HTTPStatus == http.StatusNotFound {
				return status.Error(codes.NotFound, appErr.Error())
			}
			if appErr.HTTPStatus == http.StatusConflict {
				return status.Error(codes.AlreadyExists, appErr.Error())
			}
			return status.Error(codes.Internal, "unknown error occurred")
		}
	}

	// check for specific known error types
	if errors.Is(err, ierr.ErrUserExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, ierr.ErrInvalidCredentials) {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if errors.Is(err, ierr.ErrInvalidToken) {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if errors.Is(err, ierr.ErrUserNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	// default to internal error
	return status.Error(codes.Internal, fmt.Sprintf("internal error: %v", err))
}
