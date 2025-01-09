package service

import "errors"

var (
	ErrInternal = errors.New("internal server error")
	ErrInternalTryAgainLater = errors.New("internal server error, please try again later")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidCode = errors.New("invalid code")
	ErrUserNotFound = errors.New("user not found")
)
