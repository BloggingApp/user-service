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
	AvatarURL   *string             `json:"avatar_url"`
	Bio         *string             `json:"bio"`
	Role        string              `json:"role"`
	Followers   int64               `json:"followers"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	SocialLinks []*model.SocialLink `json:"social_links"`
}

func GetUserDtoFromFullUser(fullUser model.FullUser) *GetUserDto {
	return &GetUserDto{
		ID: fullUser.ID,
		Username: fullUser.Username,
		DisplayName: fullUser.DisplayName,
		AvatarURL: fullUser.AvatarURL,
		Bio: fullUser.Bio,
		Role: fullUser.Role,
		Followers: fullUser.Followers,
		CreatedAt: fullUser.CreatedAt,
		UpdatedAt: fullUser.UpdatedAt,
		SocialLinks: fullUser.SocialLinks,
	}
}

type GetUserFollowersDto []*model.FullFollower
