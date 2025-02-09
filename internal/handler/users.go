package handler

import (
	"net/http"
	"strings"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) usersMe(c *gin.Context) {
	user := h.getUser(c)

	c.JSON(http.StatusOK, model.FullUserWithoutPasswordHashFromFullUser(*user))
}

func (h *Handler) usersGetByUsername(c *gin.Context) {
	username := c.GetString("username")

	user, err := h.services.User.FindByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, user)
}

type usersGetFollowersInput struct {
	Limit  int `json:"limit" binding:"required"`
	Offset int `json:"offset" binding:"min=0"`
}

func (h *Handler) usersGetFollowers(c *gin.Context) {
	user := h.getUser(c)

	var input usersGetFollowersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	subs, err := h.services.User.FindUserFollowers(c.Request.Context(), user.ID, input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, subs)
}

func (h *Handler) usersFollow(c *gin.Context) {
	follower := h.getUser(c)

	userIDString := strings.TrimSpace(c.Param("userID"))
	userID, err := uuid.ParseBytes([]byte(userIDString))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, errInvalidID.Error()))
		return
	}

	if err := h.services.User.Follow(c.Request.Context(), model.Follower{FollowerID: follower.ID, UserID: userID}); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

type usersGetFollowsInput struct {
	Limit  int `json:"limit" binding:"required"`
	Offset int `json:"offset" binding:"min=0"`
}

func (h *Handler) usersGetFollows(c *gin.Context) {
	user := h.getUser(c)

	var input usersGetFollowsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	subscriptions, err := h.services.User.FindUserFollows(c.Request.Context(), user.ID, input.Limit, input.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

func (h *Handler) usersUpdate(c *gin.Context) {
	user := h.getUser(c)

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.User.Update(c.Request.Context(), *user, updates); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

func (h *Handler) usersSetAvatar(c *gin.Context) {
	user := h.getUser(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.User.SetAvatar(c.Request.Context(), *user, fileHeader); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

func (h *Handler) usersAddSocialLink(c *gin.Context) {
	user := h.getUser(c)

	var input dto.AddSocialLinkRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.User.AddSocialLink(c.Request.Context(), *user, input.URL); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}

func (h *Handler) usersDeleteSocialLink(c *gin.Context) {
	user := h.getUser(c)

	var input dto.DeleteSocialLinkRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewBasicResponse(false, err.Error()))
		return
	}

	if err := h.services.User.DeleteSocialLink(c.Request.Context(), *user, input.Platform); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewBasicResponse(false, err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewBasicResponse(true, ""))
}
