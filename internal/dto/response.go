package dto

import "time"

type BasicResponse struct {
	Ok        bool      `json:"ok"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

func NewBasicResponse(ok bool, details string) BasicResponse {
	return BasicResponse{
		Ok: ok,
		Details: details,
		Timestamp: time.Now(),
	}
}

type AuthResponse struct {
	Ok          bool           `json:"ok"`
	AccessToken string         `json:"access_token"`
	User        GetUserDto     `json:"user"`
}

type RefreshResponse struct {
	Ok          bool           `json:"ok"`
	AccessToken string         `json:"access_token"`
}
