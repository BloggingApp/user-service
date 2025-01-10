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
	createUserDto.Email = strings.TrimSpace(createUserDto.Email)
	createUserDto.Username = strings.TrimSpace(strings.ToLower(createUserDto.Username))

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

func (s *userService) VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*dto.GetUserDto, *utils.JWTPair, error) {
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

	user, err := s.FindByUsername(ctx, createdUser.Username)
	if err != nil {
		return nil, nil, ErrInternal
	}

	return user, jwtPair, nil
}

func (s *userService) SendSignInCode(ctx context.Context, signInDto dto.SignInDto) error {
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

func (s *userService) VerifySignInCodeAndSignIn(ctx context.Context, code int) (*dto.GetUserDto, *utils.JWTPair, error) {
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

	user, err := s.FindByUsername(ctx, userData.Username)
	if err != nil {
		return nil, nil, err
	}

	return user, jwtPair, nil
}

func (s *userService) FindByID(ctx context.Context, id uuid.UUID) (*model.FullUser, error) {
	userCache, err := redisrepo.Get[model.FullUser](s.repo.Redis.Default, ctx, redisrepo.UserKey(id.String()))
	if err == nil {
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

	return user, nil
}

func (s *userService) FindByUsername(ctx context.Context, username string) (*dto.GetUserDto, error) {
	userCache, err := redisrepo.Get[dto.GetUserDto](s.repo.Redis.Default, ctx, redisrepo.UserByUsernameKey(username))
	if err == nil {
		return userCache, nil
	}

	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get user from redis: %s", err.Error())
		return nil, ErrInternal
	}

	user, err := s.repo.Postgres.User.FindByUsername(ctx, username)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get user from postgres: %s", err.Error())
		return nil, ErrInternal
	}
	
	userDto := dto.GetUserDtoFromFullUser(*user)

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserByUsernameKey(username), userDto, time.Hour * 3); err != nil {
		s.logger.Sugar().Errorf("failed to set user in redis: %s", err.Error())
		return nil, ErrInternal
	}

	return userDto, nil
}

func (s *userService) SearchByUsername(ctx context.Context, username string, limit int, offset int) ([]*dto.GetUserDto, error) {
	maxLimit := 10

	if limit > maxLimit {
		limit = maxLimit
	}

	searchResultsCache, err := redisrepo.GetMany[dto.GetUserDto](s.repo.Redis.Default, ctx, redisrepo.SearchResultsKey(username, limit, offset))
	if err == nil {
		return searchResultsCache, nil
	}

	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get value from redis: %s", err.Error())
		return nil, ErrInternal
	}

	searchResults, err := s.repo.Postgres.User.SearchByUsername(ctx, username, limit, offset)
	if err != nil {
		s.logger.Sugar().Errorf("failed to search users by username(%s): %s", username, err.Error())
		return nil, ErrInternal
	}

	searchResultsDto := s.convertFullUsersToGetUserDtos(searchResults)

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.SearchResultsKey(username, limit, offset), searchResultsDto, time.Minute * 5); err != nil {
		s.logger.Sugar().Errorf("failed to set value in redis: %s", err.Error())
		return nil, ErrInternal
	}

	return searchResultsDto, nil
}

func (s *userService) convertFullUsersToGetUserDtos(users []*model.FullUser) []*dto.GetUserDto {
	dtos := make([]*dto.GetUserDto, len(users))
	for i, user := range users {
		dtos[i] = dto.GetUserDtoFromFullUser(*user)
	}
	return dtos	
}

func (s *userService) RefreshTokens(ctx context.Context, refreshToken string) (*utils.JWTPair, error) {
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

	user, err := s.FindByID(ctx, userID)
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
