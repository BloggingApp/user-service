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
	AvatarHash      *string   `json:"avatar_hash"`
	Bio             *string   `json:"bio"`
	Role            string    `json:"role"`
	Subscribers     int64     `json:"subscribers"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type FullUser struct {
	ID              uuid.UUID           `json:"id"`
	Email           string              `json:"email"`
	Username        string              `json:"username"`
	PasswordHash    string              `json:"password_hash"`
	DisplayName     *string             `json:"display_name"`
	AvatarHash      *string             `json:"avatar_hash"`
	Bio             *string             `json:"bio"`
	Role            string              `json:"role"`
	Subscribers     int64               `json:"subscribers"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	SocialLinks     []*SocialLink `json:"social_links"`
}
