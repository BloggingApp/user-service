package service

import "time"

const (
	ACCESS_TOKEN_EXPIRY = time.Hour * 3
	REFRESH_TOKEN_EXPIRY = time.Hour * 24 * 7 * 2
)
