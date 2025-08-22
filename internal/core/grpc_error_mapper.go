package core

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Fancu1/phoenix-rss/internal/ierr"
)

// MapGRPCError converts gRPC status errors back to internal application errors
func MapGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC status error, return as-is
		return err
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return ierr.NewValidationError(st.Message())
	case codes.Unauthenticated:
		// check the message to determine specific error type
		msg := st.Message()
		if msg == "Username already exists" {
			return ierr.ErrUserExists.WithCause(fmt.Errorf(msg))
		}
		if msg == "Invalid credentials" {
			return ierr.ErrInvalidCredentials.WithCause(fmt.Errorf(msg))
		}
		if msg == "Invalid or expired token" {
			return ierr.ErrInvalidToken.WithCause(fmt.Errorf(msg))
		}
		return ierr.ErrUnauthorized.WithCause(fmt.Errorf(msg))
	case codes.AlreadyExists:
		return ierr.ErrUserExists.WithCause(fmt.Errorf(st.Message()))
	case codes.NotFound:
		return ierr.ErrUserNotFound.WithCause(fmt.Errorf(st.Message()))
	case codes.PermissionDenied:
		if st.Message() == "Not subscribed to this feed" {
			return ierr.ErrNotSubscribed
		}
		return ierr.ErrUnauthorized.WithCause(fmt.Errorf(st.Message()))
	case codes.Internal:
		return ierr.ErrInternalServer.WithCause(fmt.Errorf(st.Message()))
	default:
		return ierr.ErrInternalServer.WithCause(fmt.Errorf("gRPC error: %v", err))
	}
}
