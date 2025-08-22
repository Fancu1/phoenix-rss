package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheck(h *gin.Context) {
	h.JSON(http.StatusOK, gin.H{"status": "ok"})
}
