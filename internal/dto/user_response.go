package dto

import (
	"time"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
)

type GetUserDto struct {
	ID                          uuid.UUID           `json:"id"`
	Username                    string              `json:"username"`
	DisplayName                 *string             `json:"display_name"`
	AvatarURL                   *string             `json:"avatar_url"`
	Bio                         *string             `json:"bio"`
	Followers                   int64               `json:"followers"`
	CreatedAt                   time.Time           `json:"created_at"`
	UpdatedAt                   time.Time           `json:"updated_at"`
	SocialLinks                 []*model.SocialLink `json:"social_links"`
	IsFollowing                 bool                `json:"is_following"`
	NewPostNotificationsEnabled bool                `json:"new_post_notifications_enabled"`
}

func GetUserDtoFromFullUser(fullUser model.FullUser) *GetUserDto {
	return &GetUserDto{
		ID: fullUser.ID,
		Username: fullUser.Username,
		DisplayName: fullUser.DisplayName,
		AvatarURL: fullUser.AvatarURL,
		Bio: fullUser.Bio,
		Followers: fullUser.Followers,
		CreatedAt: fullUser.CreatedAt,
		UpdatedAt: fullUser.UpdatedAt,
		SocialLinks: fullUser.SocialLinks,
		IsFollowing: fullUser.IsFollowing,
		NewPostNotificationsEnabled: fullUser.NewPostNotificationsEnabled,
	}
}

type GetUserFollowersDto []*model.FullFollower
