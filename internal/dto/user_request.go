package dto

type CreateUserReq struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=8,max=48"`
}

type SignInReq struct {
	EmailOrUsername string `json:"email_or_username" binding:"required"`
	Password        string `json:"password" binding:"required,min=3,max=48"`
}

type AddSocialLinkReq struct {
	URL string `json:"url" binding:"required"`
}

type DeleteSocialLinkReq struct {
	Platform string `json:"platform" binding:"required"`
}

type UpdatePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required,min=8"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type RequestForgotPasswordCodeReq struct {
	Email string `json:"email" binding:"required,email"`
}

type ChangeForgottenPasswordReq struct {
	Code        int    `json:"code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
