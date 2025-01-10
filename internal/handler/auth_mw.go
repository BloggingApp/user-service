package handler

import (
	"net/http"
	"strings"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) authMiddleware(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, errNotAuthorized.Error()))
		c.Abort()
		return
	}

	accessToken := strings.Split(header, " ")[1]
	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, errNotAuthorized.Error()))
		c.Abort()
		return
	}

	user, err := h.getUserDataFromAccessTokenClaims(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		c.Abort()
		return
	}

	c.Set("user", *user)

	c.Next()
}
