package dto

import (
	"time"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
)

type CreateUserDto struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=8,max=48"`
}

type SignInDto struct {
	Username *string `json:"username" binding:"min=3,max=20"`
	Email    *string `json:"email" binding:"email"`
	Password string  `json:"password" binding:"required,min=3,max=48"`
}

type GetUserDto struct {
	ID           uuid.UUID           `json:"id"`
	Email        string              `json:"email"`
	Username     string              `json:"username"`
	DisplayName  *string             `json:"display_name"`
	Bio          *string             `json:"bio"`
	Role         string              `json:"role"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	SocialLinks  []*model.SocialLink `json:"social_links"`
}

type RabbitMQNotificateUserCodeDto struct {
	Email string `json:"email"`
	Code  int    `json:"code"`
}
