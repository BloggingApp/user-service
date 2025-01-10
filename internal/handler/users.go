package handler

import (
	"net/http"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) usersMe(c *gin.Context) {
	user := h.getUser(c)

	c.JSON(http.StatusFound, dto.GetUserDtoFromFullUser(*user))
}

func (h *Handler) usersGetByUsername(c *gin.Context) {
	username := c.GetString("username")

	user, err := h.services.User.FindByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err))
		return
	}

	c.JSON(http.StatusFound, user)
}
