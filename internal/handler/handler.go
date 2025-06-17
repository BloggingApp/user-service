package handler

import (
	"context"
	"os"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jwtmanager "github.com/morf1lo/jwt-pair-manager"
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
		AllowMethods: []string{"POST", "GET", "PATCH", "PUT", "DELETE"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/sign-up/send-code", h.authSendRegistrationCode)
			auth.POST("/sign-up/resend-code", h.authResendRegistrationCode)
			auth.POST("/sign-up/verify", h.authVerifyRegistrationCodeAndCreateUser)
			auth.POST("/sign-in/send-code", h.authSendSignInCode)
			auth.POST("/sign-in/resend-code", h.authSendSignInCode)
			auth.POST("/sign-in/verify", h.authVerifySignInCodeAndSignIn)
			auth.POST("/refresh", h.authRefresh)
			auth.PATCH("/update-pw", h.authMiddleware, h.authUpdatePassword)
			auth.POST("/request-fp-code", h.authRequestForgotPasswordCode)
			auth.PATCH("/change-forgotten-pw-by-code", h.authChangeForgottenPasswordByCode)
		}

		users := v1.Group("/users")
		{
			me := users.Group("/@me")
			{
				me.Use(h.authMiddleware)

				me.GET("", h.usersMe)
				me.GET("/followers", h.usersGetFollowers)
				me.GET("/follows", h.usersGetFollows)

				update := me.Group("/update")
				{
					update.PATCH("", h.usersUpdate)
					update.PATCH("/setAvatar", h.usersSetAvatar)

					socialLinks := update.Group("/socialLinks")
					{
						socialLinks.PUT("", h.usersAddSocialLink)
						socialLinks.DELETE("", h.usersDeleteSocialLink)
					}
				}
			}

			users.GET("/byUsername/:username", h.authMiddleware, h.usernameMiddleware, h.usersGetByUsername)
			users.PUT("/:userID/follow", h.authMiddleware, h.usersFollow)
			users.DELETE("/:userID/unfollow", h.authMiddleware, h.usersUnfollow)
			users.PATCH("/:userID/notifications", h.authMiddleware, h.usersUpdateNewPostNotificationsEnabled)
		}
	}

	return r
}

func (h *Handler) getUserDataFromAccessTokenClaims(ctx context.Context, accessToken string) (*model.FullUser, error) {
	claims, err := jwtmanager.DecodeJWT(accessToken, []byte(os.Getenv("ACCESS_SECRET")))
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
