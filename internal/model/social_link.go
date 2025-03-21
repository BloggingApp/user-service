package model

import "github.com/google/uuid"

type SocialLink struct {
	UserID   uuid.UUID `json:"user_id"`
	Platform string    `json:"platform"`
	URL      string    `json:"url"`
}
