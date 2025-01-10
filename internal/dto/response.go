package dto

import "time"

type BasicResponse struct {
	Ok        bool      `json:"ok"`
	Error     error     `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

func NewBasicResponse(ok bool, err error) BasicResponse {
	return BasicResponse{
		Ok: ok,
		Error: err,
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
