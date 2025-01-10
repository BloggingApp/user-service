package handler

import (
	"github.com/BloggingApp/user-service/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	services *service.Service
}

func New(services *service.Service) *Handler {
	return &Handler{
		services: services,
	}
}

func InitRoutes() *gin.Engine {
	r := gin.New()

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/", )
		}
	}

	return r
}
