package handler

import (
	"net/http"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) authSendRegistrationCode(c *gin.Context) {
	var input dto.CreateUserDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err))
		return
	}

	if err := h.services.User.SendRegistrationCode(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, nil))
}

type verifyRegistrationCodeInput struct {
	Code int `json:"code" binding:"required"`
}

func (h *Handler) authVerifyRegistrationCodeAndCreateUser(c *gin.Context) {
	var input verifyRegistrationCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err))
		return
	}

	user, tokenPair, err := h.services.User.VerifyRegistrationCodeAndCreateUser(c.Request.Context(), input.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err))
		return
	}

	c.SetCookie("refresh_token", tokenPair.RefreshToken, int(tokenPair.RefreshTokenExp.Unix()), "/", "localhost", true, true)

	c.JSON(http.StatusCreated, dto.AuthResponse{Ok: true, AccessToken: tokenPair.AccessToken, User: *user})
}
