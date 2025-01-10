package dto

import (
	"time"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
)

type GetUserDto struct {
	ID          uuid.UUID           `json:"id"`
	Username    string              `json:"username"`
	DisplayName *string             `json:"display_name"`
	AvatarHash  *string             `json:"avatar_hash"`
	Bio         *string             `json:"bio"`
	Role        string              `json:"role"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	SocialLinks []*model.SocialLink `json:"social_links"`
}
