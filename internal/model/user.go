package model

import (
	"time"

	"github.com/google/uuid"
)

type TempUserData struct {
	Email        string `json:"email"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	PasswordHash    string    `json:"password_hash"`
	DisplayName     *string   `json:"display_name"`
	AvatarURL       *string   `json:"avatar_url"`
	Bio             *string   `json:"bio"`
	Role            string    `json:"role"`
	Followers       int64     `json:"followers"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type FullUser struct {
	ID                          uuid.UUID     `json:"id"`
	Email                       string        `json:"email"`
	Username                    string        `json:"username"`
	DisplayName                 *string       `json:"display_name"`
	AvatarURL                   *string       `json:"avatar_url"`
	Bio                         *string       `json:"bio"`
	Role                        string        `json:"role"`
	Followers                   int64         `json:"followers"`
	CreatedAt                   time.Time     `json:"created_at"`
	UpdatedAt                   time.Time     `json:"updated_at"`
	SocialLinks                 []*SocialLink `json:"social_links"`
	IsFollowing                 bool          `json:"is_following"`
	NewPostNotificationsEnabled bool          `json:"new_post_notifications_enabled"`
}
