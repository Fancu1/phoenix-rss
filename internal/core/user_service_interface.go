package core

import (
	"github.com/golang-jwt/jwt/v5"

	"github.com/Fancu1/phoenix-rss/internal/models"
)

// UserServiceInterface define the contract for user service operations
type UserServiceInterface interface {
	Register(username, password string) (*models.User, error)
	Login(username, password string) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetUserFromToken(tokenString string) (*models.User, error)
}
