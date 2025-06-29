package handler

import (
	"net/http"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/gin-gonic/gin"
)

func (h *Handler) authSendRegistrationCode(c *gin.Context) {
	var input dto.CreateUserReq
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Auth.SendRegistrationCode(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

func (h *Handler) authResendRegistrationCode(c *gin.Context) {
	var input dto.CreateUserReq
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Auth.ResendRegistrationCode(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

type authVerifyRegistrationCodeInput struct {
	Code int `json:"code" binding:"required"`
}

func (h *Handler) authVerifyRegistrationCodeAndCreateUser(c *gin.Context) {
	var input authVerifyRegistrationCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	user, tokenPair, err := h.services.Auth.VerifyRegistrationCodeAndCreateUser(c.Request.Context(), input.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.SetCookie("refresh_token", tokenPair.RefreshToken, int(tokenPair.RefreshTokenExp.Seconds()), "/", "localhost", true, true)

	c.JSON(http.StatusCreated, dto.AuthResponse{Ok: true, AccessToken: tokenPair.AccessToken, User: *user})
}

func (h *Handler) authSendSignInCode(c *gin.Context) {
	var input dto.SignInReq
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Auth.SendSignInCode(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

type authVerifySignInCodeInput struct {
	Code int `json:"code" binding:"required"`
}

func (h *Handler) authVerifySignInCodeAndSignIn(c *gin.Context) {
	var input authVerifySignInCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	user, tokenPair, err := h.services.Auth.VerifySignInCodeAndSignIn(c.Request.Context(), input.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.SetCookie("refresh_token", tokenPair.RefreshToken, int(tokenPair.RefreshTokenExp.Seconds()), "/", "localhost", true, true)

	c.JSON(http.StatusCreated, dto.AuthResponse{Ok: true, AccessToken: tokenPair.AccessToken, User: *user})
}

func (h *Handler) authRefresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, err.Error()))
		return
	}

	tokenPair, err := h.services.Auth.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.SetCookie("refresh_token", tokenPair.RefreshToken, int(tokenPair.RefreshTokenExp.Seconds()), "/", "localhost", true, true)

	c.JSON(http.StatusCreated, dto.RefreshResponse{Ok: true, AccessToken: tokenPair.AccessToken})
}

func (h *Handler) authUpdatePassword(c *gin.Context) {
	user := h.getUser(c)

	var input dto.UpdatePasswordReq
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Auth.UpdatePassword(c.Request.Context(), user.ID, input.OldPassword, input.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) authRequestForgotPasswordCode(c *gin.Context) {
	var input dto.RequestForgotPasswordCodeReq
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Auth.RequestForgotPasswordCode(c.Request.Context(), input.Email); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

func (h *Handler) authChangeForgottenPasswordByCode(c *gin.Context) {
	var input dto.ChangeForgottenPasswordReq
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.Auth.ChangeForgottenPasswordByCode(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}
