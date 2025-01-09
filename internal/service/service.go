package service

import (
	"context"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/pkg/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type User interface {
	GenerateRegistrationCode(ctx context.Context, createUserDto *dto.CreateUserDto) error
	VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*utils.JWTPair, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type Service struct {
	User
}

func New(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) *Service {
	return &Service{
		User: newUserService(logger, repo, rabbitmq),
	}
}
