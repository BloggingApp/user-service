package handler

import (
	"context"
	"os"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/service"
	"github.com/BloggingApp/user-service/pkg/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Handler struct {
	services *service.Service
}

func New(services *service.Service) *Handler {
	return &Handler{
		services: services,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{viper.GetString("client.origin")},
		AllowMethods: []string{"POST", "GET", "PATCH"},
		AllowCredentials: true,
	}))

	r.Static("/public", "./public")

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/sign-up/send-code", h.authSendRegistrationCode)
			auth.POST("/sign-up/verify", h.authVerifyRegistrationCodeAndCreateUser)
			auth.POST("/sign-in/send-code", h.authSendSignInCode)
			auth.POST("/sign-in/verify", h.authVerifySignInCodeAndSignIn)
			auth.POST("/refresh", h.authRefresh)
		}

		users := v1.Group("/users")
		{
			me := users.Group("/@me")
			{
				me.Use(h.authMiddleware)

				me.GET("", h.usersMe)
				me.GET("/subscribers", h.usersGetSubscribers)
				me.GET("/subscriptions", h.usersGetSubscriptions)

				update := me.Group("/update")
				{
					update.PATCH("", h.usersUpdate)
					update.PATCH("/setAvatar", h.usersSetAvatar)
					update.PUT("/addSocialLink", h.usersAddSocialLink)
				}
			}

			users.GET("/byUsername/:username", h.authMiddleware, h.usernameMiddleware, h.usersGetByUsername)
			users.PUT("/subscribe/:userID", h.authMiddleware, h.usersSubscribe)
		}
	}

	return r
}

func (h *Handler) getUserDataFromAccessTokenClaims(ctx context.Context, accessToken string) (*model.FullUser, error) {
	claims, err := utils.DecodeJWT(accessToken, []byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}

	idString := claims["id"].(string)
	id, err := uuid.Parse(idString)
	if err != nil {
		return nil, err
	}

	user, err := h.services.User.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *Handler) getUser(c *gin.Context) *model.FullUser {
	userReq, _ := c.Get("user")

	user, ok := userReq.(model.FullUser)
	if !ok {
		return nil
	}

	return &user
}
