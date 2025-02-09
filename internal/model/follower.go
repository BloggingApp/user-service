package model

import "github.com/google/uuid"

type Follower struct {
	UserID      uuid.UUID `json:"user_id"`
	FollowerID  uuid.UUID `json:"follower_id"`
}

type FullFollower struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName *string   `json:"display_name"`
	AvatarHash  *string   `json:"avatar_hash"`
	Bio         *string   `json:"bio"`
}
