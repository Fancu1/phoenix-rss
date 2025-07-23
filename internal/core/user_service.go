package core

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Fancu1/phoenix-rss/internal/ierr"
	"github.com/Fancu1/phoenix-rss/internal/models"
	"github.com/Fancu1/phoenix-rss/internal/repository"
)

type UserServiceInterface interface {
	Register(username, password string) (*models.User, error)
	Login(username, password string) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetUserFromToken(tokenString string) (*models.User, error)
}

type UserService struct {
	userRepo  *repository.UserRepository
	jwtSecret []byte
}

func NewUserService(userRepo *repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *UserService) Register(username, password string) (*models.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to check existing user '%s': %w", username, err))
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user '%s' already exists: %w", username, ierr.ErrUserExists)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ierr.NewInternalError(fmt.Errorf("failed to hash password for user '%s': %w", username, err))
	}

	// Create user
	user := &models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	createdUser, err := s.userRepo.Create(user)
	if err != nil {
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to create user '%s': %w", username, err))
	}

	return createdUser, nil
}

func (s *UserService) Login(username, password string) (string, error) {
	// Get user
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", ierr.NewDatabaseError(fmt.Errorf("failed to get user '%s': %w", username, err))
	}
	if user == nil {
		return "", fmt.Errorf("login failed for user '%s': %w", username, ierr.ErrInvalidCredentials)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", fmt.Errorf("password verification failed for user '%s': %w", username, ierr.ErrInvalidCredentials)
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", ierr.NewInternalError(fmt.Errorf("failed to generate token for user '%s' (ID: %d): %w", username, user.ID, err))
	}

	return tokenString, nil
}

func (s *UserService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", ierr.ErrInvalidToken.WithCause(err))
	}

	if !token.Valid {
		return nil, fmt.Errorf("token validation failed: %w", ierr.ErrInvalidToken)
	}

	return token, nil
}

func (s *UserService) GetUserFromToken(tokenString string) (*models.User, error) {
	token, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err // Already wrapped with context in ValidateToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims structure: %w", ierr.ErrInvalidToken)
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid user_id in token claims: %w", ierr.ErrInvalidToken)
	}

	userID := uint(userIDFloat)
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, ierr.NewDatabaseError(fmt.Errorf("failed to get user by ID %d from token: %w", userID, err))
	}
	if user == nil {
		return nil, fmt.Errorf("user with ID %d not found (from token): %w", userID, ierr.ErrUserNotFound)
	}

	return user, nil
}
