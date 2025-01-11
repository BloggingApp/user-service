package model

import "github.com/google/uuid"

type Subscriber struct {
	UserID uuid.UUID `json:"user_id"`
	SubID  uuid.UUID `json:"sub_id"`
}

type FullSub struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName *string   `json:"display_name"`
	AvatarHash  *string   `json:"avatar_hash"`
	Bio         *string   `json:"bio"`
}
