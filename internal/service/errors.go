package service

import "errors"

var (
	ErrInternal = errors.New("internal server error")
	ErrUsernameCannotContainSpecialCharacters = errors.New("username cannot contain special characters")
	ErrInternalTryAgainLater = errors.New("internal server error, please try again later")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidCode = errors.New("invalid code")
	ErrUserNotFound = errors.New("user not found")
	ErrUnauthorized = errors.New("user is not authorized")
	ErrSubToYourself = errors.New("you cannot subscribe to yourself")
	ErrAlreadySubscribed = errors.New("you have already subscribed to this user")
	ErrCooldown = errors.New("cooldown")
	ErrFieldsNotAllowedToUpdate = errors.New("these fields are not allowed to be updated")
	ErrUserWithUsernameAlreadyExists = errors.New("user with this username is already exists")
	ErrUserWithEmailAlreadyExists = errors.New("user with this email is already exists")
	ErrUserAlreadyExists = errors.New("user with this email or username is already exists")
)
