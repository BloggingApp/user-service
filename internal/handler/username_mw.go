package handler

import (
	"net/http"
	"strings"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) usernameMiddleware(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errUsernameIsNotProvided.Error()))
		c.Abort()
		return
	}

	if !strings.HasPrefix(username, "@") {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errInvalidUsername.Error()))
		c.Abort()
		return
	}

	extractedUsername := strings.TrimSpace(strings.Split(username, "@")[0])
	if extractedUsername == "" {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errUsernameIsNotProvided.Error()))
		c.Abort()
		return
	}

	c.Set("username", extractedUsername)

	c.Next()
}
