package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	PasswordHash    string    `json:"password_hash"`
	DisplayName     *string   `json:"display_name"`
	AvatarURL       *string   `json:"avatar_url"`
	Bio             *string   `json:"bio"`
	Role            string    `json:"role"`
	Subscribers     int64     `json:"subscribers"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type FullUser struct {
	ID           uuid.UUID           `json:"id"`
	Email        string              `json:"email"`
	Username     string              `json:"username"`
	PasswordHash string              `json:"password_hash"`
	DisplayName  *string             `json:"display_name"`
	AvatarURL    *string             `json:"avatar_url"`
	Bio          *string             `json:"bio"`
	Role         string              `json:"role"`
	Subscribers  int64               `json:"subscribers"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	SocialLinks  []*SocialLink       `json:"social_links"`
}

type FullUserWithoutPasswordHash struct {
	ID           uuid.UUID           `json:"id"`
	Email        string              `json:"email"`
	Username     string              `json:"username"`
	DisplayName  *string             `json:"display_name"`
	AvatarURL    *string             `json:"avatar_url"`
	Bio          *string             `json:"bio"`
	Role         string              `json:"role"`
	Subscribers  int64               `json:"subscribers"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	SocialLinks  []*SocialLink       `json:"social_links"`
}

func FullUserWithoutPasswordHashFromFullUser(fullUser FullUser) FullUserWithoutPasswordHash  {
	return FullUserWithoutPasswordHash{
		ID: fullUser.ID,
		Email: fullUser.Email,
		Username: fullUser.Username,
		DisplayName: fullUser.DisplayName,
		AvatarURL: fullUser.AvatarURL,
		Bio: fullUser.Bio,
		Role: fullUser.Role,
		Subscribers: fullUser.Subscribers,
		CreatedAt: fullUser.CreatedAt,
		UpdatedAt: fullUser.UpdatedAt,
		SocialLinks: fullUser.SocialLinks,
	}
}
