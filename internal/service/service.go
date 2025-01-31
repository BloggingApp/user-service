package service

import (
	"context"
	"mime/multipart"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Auth interface {
	SendRegistrationCode(ctx context.Context, createUserDto dto.CreateUserDto) error
	VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*dto.GetUserDto, *utils.JWTPair, error)
	SendSignInCode(ctx context.Context, signInDto dto.SignInDto) error
	VerifySignInCodeAndSignIn(ctx context.Context, code int) (*dto.GetUserDto, *utils.JWTPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*utils.JWTPair, error)
}

type User interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.FullUser, error)
	FindByUsername(ctx context.Context, username string) (*dto.GetUserDto, error)
	SearchByUsername(ctx context.Context, username string, limit int, offset int) ([]*dto.GetUserDto, error)
	FindUserSubscribers(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullSub, error)
	Subscribe(ctx context.Context, subscriber model.Subscriber) error
	FindUserSubscriptions(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullSub, error)
	Update(ctx context.Context, user model.FullUser, updates map[string]interface{}) error
	SetAvatar(ctx context.Context, user model.FullUser, fileHeader *multipart.FileHeader) error
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
