package service

import (
	"context"
	"encoding/json"
	"os"
	"strings"
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

type authService struct {
	logger *zap.Logger
	repo *repository.Repository
	rabbitmq *rabbitmq.MQConn
	userService User
}

func newAuthService(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn, userService User) Auth {
	return &authService{
		logger: logger,
		repo: repo,
		rabbitmq: rabbitmq,
		userService: userService,
	}
}

func (s *authService) SendRegistrationCode(ctx context.Context, createUserDto dto.CreateUserDto) error {
	createUserDto.Email = strings.TrimSpace(createUserDto.Email)
	createUserDto.Username = strings.TrimSpace(strings.ToLower(createUserDto.Username))

	if strings.ContainsAny(createUserDto.Username, "!@#№$;%^:&?*()-/\\|,<>`~+= ") {
		return ErrUsernameCannotContainSpecialCharacters
	}

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

func (s *authService) VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*dto.GetUserDto, *utils.JWTPair, error) {
	// Verifying if code exists
	redisKey := redisrepo.TempRegistrationCodeKey(code)
	userData, err := redisrepo.Get[dto.CreateUserDto](s.repo.Redis.Default, ctx, redisKey)
	if err != nil {
		if err == redis.Nil {
			return nil, nil, ErrInvalidCode
		}

		s.logger.Sugar().Errorf("failed to get value with key(%s) from redis: %s", redisKey, err.Error())
		return nil, nil, ErrInternal
	}

	if err := s.repo.Redis.Default.Del(ctx, redisKey).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete value with key(%s) from redis: %s", redisKey, err.Error())
		return nil, nil, ErrInternal
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
	if err != nil {
		s.logger.Sugar().Errorf("failed to generate password hash: %s", err.Error())
		return nil, nil, ErrInternal
	}

	newUser := model.User{
		Email: userData.Email,
		Username: userData.Username,
		PasswordHash: string(passwordHash),
	}
	createdUser, err := s.repo.Postgres.User.Create(ctx, newUser)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user in postgres: %s", err.Error())
		return nil, nil, ErrInternal
	}

	jwtPair, err := utils.GenerateJWTPair(utils.GenerateJWTPairDto{
		Method: jwt.SigningMethodHS256,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": createdUser.ID.String(),
			"role": createdUser.Role,
		},
		AccessExpiry: ACCESS_TOKEN_EXPIRY,
		RefreshSecret: []byte(os.Getenv("REFRESH_SECRET")),
		RefreshClaims: jwt.MapClaims{
			"id": createdUser.ID.String(),
		},
		RefreshExpiry: REFRESH_TOKEN_EXPIRY,
	})
	if err != nil {
		s.logger.Sugar().Fatalf("failed to generate jwt pair: %s", err.Error())
		return nil, nil, ErrInternal
	}

	user, err := s.userService.FindByUsername(ctx, createdUser.Username)
	if err != nil {
		return nil, nil, ErrInternal
	}

	return user, jwtPair, nil
}

func (s *authService) SendSignInCode(ctx context.Context, signInDto dto.SignInDto) error {
	signInDto.EmailOrUsername = strings.ToLower(signInDto.EmailOrUsername)

	user, err := s.repo.Postgres.User.FindByEmailOrUsername(ctx, signInDto.EmailOrUsername, signInDto.EmailOrUsername)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrInvalidCredentials
		}

		s.logger.Sugar().Errorf("failed to get user(email: %s or username: %s) from postgres: %s", signInDto.EmailOrUsername, signInDto.EmailOrUsername, err.Error())
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

func (s *authService) VerifySignInCodeAndSignIn(ctx context.Context, code int) (*dto.GetUserDto, *utils.JWTPair, error) {
	// Verifying if code exists
	redisKey := redisrepo.TempSignInCodeKey(code)
	userData, err := redisrepo.Get[model.User](s.repo.Redis.Default, ctx, redisKey)
	if err != nil {
		if err == redis.Nil {
			return nil, nil, ErrInvalidCode
		}

		s.logger.Sugar().Errorf("failed to get value with key(%s) from redis: %s", redisKey, err.Error())
		return nil, nil, ErrInternal
	}

	if err := s.repo.Redis.Default.Del(ctx, redisKey).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete value with key(%s) from redis: %s", redisKey, err.Error())
		return nil, nil, ErrInternal
	}

	jwtPair, err := utils.GenerateJWTPair(utils.GenerateJWTPairDto{
		Method: jwt.SigningMethodHS256,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": userData.ID.String(),
			"role": userData.Role,
		},
		AccessExpiry: ACCESS_TOKEN_EXPIRY,
		RefreshSecret: []byte(os.Getenv("REFRESH_SECRET")),
		RefreshClaims: jwt.MapClaims{
			"id": userData.ID.String(),
		},
		RefreshExpiry: REFRESH_TOKEN_EXPIRY,
	})
	if err != nil {
		s.logger.Sugar().Fatalf("failed to generate jwt pair: %s", err.Error())
		return nil, nil, ErrInternal
	}

	user, err := s.userService.FindByUsername(ctx, userData.Username)
	if err != nil {
		return nil, nil, err
	}

	return user, jwtPair, nil
}

func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*utils.JWTPair, error) {
	decodedToken, err := utils.DecodeJWT(refreshToken, []byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, ErrUnauthorized
	}

	id, exists := decodedToken["id"].(string)
	if !exists {
		return nil, ErrUnauthorized
	}

	userID, err := uuid.ParseBytes([]byte(id))
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := s.userService.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	jwtPair, err := utils.GenerateJWTPair(utils.GenerateJWTPairDto{
		Method: jwt.SigningMethodHS256,
		AccessExpiry: ACCESS_TOKEN_EXPIRY,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": user.ID.String(),
			"role": user.Role,
			"exp": time.Hour * 3,
		},
		RefreshExpiry: REFRESH_TOKEN_EXPIRY,
		RefreshSecret: []byte(os.Getenv("REFRESH_SECRET")),
		RefreshClaims: jwt.MapClaims{
			"id": user.ID.String(),
			"exp": time.Hour * 24 * 7 * 2,
		},
	})
	if err != nil {
		s.logger.Sugar().Fatalf("failed to generate jwt pair: %s", err.Error())
		return nil, ErrInternal
	}

	return jwtPair, nil
}
