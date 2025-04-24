package service

import (
	"context"
	"mime/multipart"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/google/uuid"
	jwtmanager "github.com/morf1lo/jwt-pair-manager"
	"go.uber.org/zap"
)

type Auth interface {
	SendRegistrationCode(ctx context.Context, createUserDto dto.CreateUser) error
	VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*dto.GetUserDto, *jwtmanager.JWTPair, error)
	SendSignInCode(ctx context.Context, signInDto dto.SignIn) error
	VerifySignInCodeAndSignIn(ctx context.Context, code int) (*dto.GetUserDto, *jwtmanager.JWTPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*jwtmanager.JWTPair, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
	RequestForgotPasswordCode(ctx context.Context, email string) error
}

type User interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.FullUser, error)
	FindByUsername(ctx context.Context, getterID *uuid.UUID, username string) (*dto.GetUserDto, error)
	SearchByUsername(ctx context.Context, username string, limit int, offset int) ([]*dto.GetUserDto, error)
	FindUserFollowers(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error)
	Follow(ctx context.Context, follower model.Follower) error
	Unfollow(ctx context.Context, follower model.Follower) error
	UpdateNewPostNotificationsEnabled(ctx context.Context, follower model.Follower) error
	FindUserFollows(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error)
	Update(ctx context.Context, user model.FullUser, updates map[string]interface{}) error
	SetAvatar(ctx context.Context, user model.FullUser, fileHeader *multipart.FileHeader) error
	AddSocialLink(ctx context.Context, user model.FullUser, link string) error
	DeleteSocialLink(ctx context.Context, user model.FullUser, platform string) error
}

type Service struct {
	Auth
	User
}

func New(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) *Service {
	userService := newUserService(logger, repo, rabbitmq)

	return &Service{
		Auth: newAuthService(logger, repo, rabbitmq, userService),
		User: userService,
	}
}
