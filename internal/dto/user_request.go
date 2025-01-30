package dto

type CreateUser struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=8,max=48"`
}

type SignIn struct {
	EmailOrUsername string `json:"email_or_username" binding:"required"`
	Password        string `json:"password" binding:"required,min=3,max=48"`
}

type AddSocialLinkRequest struct {
	Link string `json:"link" binding:"required"`
}

type DeleteSocialLinkRequest struct {
	Platform string `json:"platform" binding:"required"`
}
