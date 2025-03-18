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
	ErrFollowToYourself = errors.New("you cannot follow to yourself")
	ErrAlreadyFollowing = errors.New("you are already following this user")
	ErrCooldown = errors.New("cooldown")
	ErrFieldsNotAllowedToUpdate = errors.New("these fields are not allowed to be updated")
	ErrUserWithUsernameAlreadyExists = errors.New("user with this username is already exists")
	ErrUserWithEmailAlreadyExists = errors.New("user with this email is already exists")
	ErrUserAlreadyExists = errors.New("user with this email or username is already exists")
	ErrFileMustBeImage = errors.New("file must be image")
	ErrFileMustHaveValidExtension = errors.New("file must have a valid extension")
	ErrFailedToUploadAvatarToCDN = errors.New("failed to upload avatar to cdn")
	ErrMaxSocialLinksAchieved = errors.New("maximum count of social links achieved")
	ErrLinkHasInvalidType = errors.New("the link has invalid type")
)
