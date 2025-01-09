package service

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/internal/repository/redisrepo"
	"github.com/BloggingApp/user-service/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	logger *zap.Logger
	repo *repository.Repository
	rabbitmq *rabbitmq.MQConn
}

const (
	MIN_REGISTRATION_CODE = 100_000
	MAX_REGISTRATION_CODE = 999_999

	MIN_SIGNIN_CODE = 100_000
	MAX_SIGNIN_CODE = 999_999
)

func newUserService(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) User {
	return &userService{
		logger: logger,
		repo: repo,
		rabbitmq: rabbitmq,
	}
}

func (s *userService) SendRegistrationCode(ctx context.Context, createUserDto dto.CreateUserDto) error {
	// Trying to generate unique registration code
	code := 0
	maxAttempts := 10
	for i := 1; i <= maxAttempts; i++ {
		code = utils.NewRandomCode(MIN_REGISTRATION_CODE, MAX_REGISTRATION_CODE)
		_, err := redisrepo.Get[dto.CreateUserDto](s.repo.Redis.Default, ctx, redisrepo.TempRegistrationCodeKey(code))
		if err == redis.Nil {
			break
		}
		if err != nil {
			s.logger.Sugar().Errorf("failed to get value from redis: %s", err.Error())
			return ErrInternal
		}
		if i == maxAttempts {
			return ErrInternalTryAgainLater
		}
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.TempRegistrationCodeKey(code), createUserDto, time.Hour * 3); err != nil {
		s.logger.Sugar().Errorf("failed to set temporary user in redis: %s", err.Error())
		return ErrInternal
	}

	queueData, err := json.Marshal(&dto.RabbitMQNotificateUserCodeDto{
		Email: createUserDto.Email,
		Code: code,
	})
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal json: %s", err.Error())
		return ErrInternal
	}

	if err := s.rabbitmq.Publish(rabbitmq.REGISTRATION_CODE_MAIL_QUEUE, queueData); err != nil {
		s.logger.Sugar().Errorf("failed to publish to rabbitmq queue(%s): %s", rabbitmq.REGISTRATION_CODE_MAIL_QUEUE, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*utils.JWTPair, error) {
	// Verifying if code exists
	key := redisrepo.TempRegistrationCodeKey(code)
	userData, err := redisrepo.Get[dto.CreateUserDto](s.repo.Redis.Default, ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, ErrInvalidCode
		}

		s.logger.Sugar().Errorf("failed to get value with key(%s) from redis: %s", key, err.Error())
		return nil, ErrInternal
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
	if err != nil {
		s.logger.Sugar().Errorf("failed to generate password hash: %s", err.Error())
		return nil, ErrInternal
	}

	user := model.User{
		Email: userData.Email,
		Username: userData.Username,
		PasswordHash: string(passwordHash),
	}
	createdUser, err := s.repo.Postgres.User.Create(ctx, user)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user in postgres: %s", err.Error())
		return nil, ErrInternal
	}

	jwtPair, err := utils.GenerateJWTPair(utils.GenerateJWTPairDto{
		Method: jwt.SigningMethodHS256,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": createdUser.ID.String(),
			"exp": time.Hour * 3,
		},
		RefreshSecret: []byte(os.Getenv("REFRESH_SECRET")),
		RefreshClaims: jwt.MapClaims{
			"id": createdUser.ID.String(),
			"exp": time.Hour * 24 * 7 * 2,
		},
	})
	if err != nil {
		s.logger.Sugar().Fatalf("failed to generate jwt pair: %s", err.Error())
		return nil, ErrInternal
	}

	return jwtPair, nil
}

func (s *userService) SendSignInCode(ctx context.Context, signInDto dto.SignInDto) error {
	user, err := s.repo.Postgres.User.FindByEmailOrUsername(ctx, *signInDto.Email, *signInDto.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrInvalidCredentials
		}

		s.logger.Sugar().Errorf("failed to get user(email: %s or username: %s) from postgres: %s", *signInDto.Email, *signInDto.Username, err.Error())
		return ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(signInDto.Password)); err != nil {
		return ErrInvalidCredentials
	}

	// Trying to generate unique sign-in code
	code := 0
	maxAttempts := 10
	for i := 1; i <= maxAttempts; i++ {
		code = utils.NewRandomCode(MIN_SIGNIN_CODE, MAX_SIGNIN_CODE)
		_, err := redisrepo.Get[model.User](s.repo.Redis.Default, ctx, redisrepo.TempSignInCodeKey(code))
		if err == redis.Nil {
			break
		}
		if err != nil {
			s.logger.Sugar().Errorf("failed to get value from redis: %s", err.Error())
			return ErrInternal
		}
		if i == maxAttempts {
			return ErrInternalTryAgainLater
		}
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.TempSignInCodeKey(code), user, time.Hour * 3); err != nil {
		s.logger.Sugar().Errorf("failed to set temporary user in redis: %s", err.Error())
		return ErrInternal
	}

	queueData, err := json.Marshal(&dto.RabbitMQNotificateUserCodeDto{
		Email: user.Email,
		Code: code,
	})
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal json: %s", err.Error())
		return ErrInternal
	}

	if err := s.rabbitmq.Publish(rabbitmq.SIGNIN_CODE_MAIL_QUEUE, queueData); err != nil {
		s.logger.Sugar().Errorf("failed to publish to rabbitmq queue(%s): %s", rabbitmq.SIGNIN_CODE_MAIL_QUEUE, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) VerifySignInCodeAndSignIn(ctx context.Context, code int) (*utils.JWTPair, error) {
	// Verifying if code exists
	key := redisrepo.TempSignInCodeKey(code)
	userData, err := redisrepo.Get[model.User](s.repo.Redis.Default, ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, ErrInvalidCode
		}

		s.logger.Sugar().Errorf("failed to get value with key(%s) from redis: %s", key, err.Error())
		return nil, ErrInternal
	}

	jwtPair, err := utils.GenerateJWTPair(utils.GenerateJWTPairDto{
		Method: jwt.SigningMethodHS256,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": userData.ID.String(),
			"exp": time.Hour * 3,
		},
		RefreshSecret: []byte(os.Getenv("REFRESH_SECRET")),
		RefreshClaims: jwt.MapClaims{
			"id": userData.ID.String(),
			"exp": time.Hour * 24 * 7 * 2,
		},
	})
	if err != nil {
		s.logger.Sugar().Fatalf("failed to generate jwt pair: %s", err.Error())
		return nil, ErrInternal
	}

	return jwtPair, nil
}

func (s *userService) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	userCache, err := redisrepo.Get[model.User](s.repo.Redis.Default, ctx, redisrepo.UserKey(id.String()))
	if err == nil {
		s.logger.Info("HELLO USER FROM REDIS")
		return userCache, nil
	}

	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get user(%s) from redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	user, err := s.repo.Postgres.FindByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrUserNotFound
		}

		s.logger.Sugar().Errorf("failed to find user(%s) from postgres: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.SetJSON(ctx, redisrepo.UserKey(id.String()), user, time.Hour * 3); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) in redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	s.logger.Info("HELLO USER FROM POSTGRES")
	return user, nil
}
