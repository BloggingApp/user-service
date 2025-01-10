package handler

import "errors"

var (
	errNotAuthorized = errors.New("user is not authorized")
	errUsernameIsNotProvided = errors.New("please provide username")
	errInvalidUsername = errors.New("invalid username, it should start with: '@'")
)
