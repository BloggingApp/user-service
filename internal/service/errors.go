package service

import "errors"

var (
	ErrInternal = errors.New("internal server error")
	ErrInvalidCode = errors.New("invalid code")
	ErrUserNotFound = errors.New("user not found")
)
