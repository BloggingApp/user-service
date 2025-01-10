package dto

type CreateUserDto struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=8,max=48"`
}

type SignInDto struct {
	Username *string `json:"username" binding:"min=3,max=20"`
	Email    *string `json:"email" binding:"email"`
	Password string  `json:"password" binding:"required,min=3,max=48"`
}
