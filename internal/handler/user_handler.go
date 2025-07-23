package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Fancu1/phoenix-rss/internal/core"
	"github.com/Fancu1/phoenix-rss/internal/ierr"
)

type UserHandler struct {
	userService core.UserServiceInterface
}

func NewUserHandler(userService core.UserServiceInterface) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(ierr.NewValidationError(err.Error()))
		return
	}

	// Basic validation
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 {
		c.Error(ierr.NewValidationError("username must be at least 3 characters"))
		return
	}
	if len(req.Password) < 6 {
		c.Error(ierr.NewValidationError("password must be at least 6 characters"))
		return
	}

	user, err := h.userService.Register(req.Username, req.Password)
	if err != nil {
		c.Error(err)
		return
	}

	// Generate token for immediate login
	token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		c.Error(ierr.NewInternalError(err))
		return
	}

	response := AuthResponse{
		Token: token,
	}
	response.User.ID = user.ID
	response.User.Username = user.Username

	c.JSON(http.StatusCreated, response)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(ierr.NewValidationError(err.Error()))
		return
	}

	token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		c.Error(err)
		return
	}

	// Get user details for response
	user, err := h.userService.GetUserFromToken(token)
	if err != nil {
		c.Error(err)
		return
	}

	response := AuthResponse{
		Token: token,
	}
	response.User.ID = user.ID
	response.User.Username = user.Username

	c.JSON(http.StatusOK, response)
}
