package service

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/internal/repository/redisrepo"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	jwtmanager "github.com/morf1lo/jwt-pair-manager"
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

func newRandomCode(min int, max int) int {
	return rand.Intn(max - min) + min
}

func (s *authService) SendRegistrationCode(ctx context.Context, createUserDto dto.CreateUserReq) error {
	createUserDto.Email = strings.TrimSpace(createUserDto.Email)
	createUserDto.Username = strings.TrimSpace(strings.ToLower(createUserDto.Username))

	if strings.ContainsAny(createUserDto.Username, "!@#№$;%^:&?*()-/\\|,<>`~+= ") {
		return ErrUsernameCannotContainSpecialCharacters
	}

	// Checking users, who are already in the registration process with these emails and usernames
	prepareEmailExists, err := s.repo.Redis.Default.Get(ctx, redisrepo.PrepareUserEmailKey(createUserDto.Email)).Bool()
	if err != nil && err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get prepare user email(%s) from redis: %s", createUserDto.Email, err.Error())
		return ErrInternal
	}
	if prepareEmailExists {
		return ErrUserWithEmailAlreadyExists
	}

	prepareUsernameExists, err := s.repo.Redis.Default.Get(ctx, redisrepo.PrepareUsernameKey(createUserDto.Username)).Bool()
	if err != nil && err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get prepare user username(%s) from redis: %s", createUserDto.Username, err.Error())
		return ErrInternal
	}
	if prepareUsernameExists {
		return ErrUserWithUsernameAlreadyExists
	}

	user, err := s.repo.Postgres.User.FindByEmailOrUsername(ctx, createUserDto.Email, createUserDto.Username)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if user != nil {
		return ErrUserAlreadyExists
	}

	// Trying to generate unique registration code
	code := 0
	maxAttempts := 10
	for i := 1; i <= maxAttempts; i++ {
		code = newRandomCode(MIN_REGISTRATION_CODE, MAX_REGISTRATION_CODE)
		_, err := redisrepo.Get[dto.CreateUserReq](s.repo.Redis.Default, ctx, redisrepo.TempRegistrationCodeKey(code))
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
	if err := s.repo.Redis.Default.Set(ctx, redisrepo.PrepareUserEmailKey(createUserDto.Email), true, time.Hour * 3); err != nil {
		s.logger.Sugar().Errorf("failed to set prepare user email(%s): %s", createUserDto.Email, err.Error())
		return ErrInternal
	}
	if err := s.repo.Redis.Default.Set(ctx, redisrepo.PrepareUsernameKey(createUserDto.Username), true, time.Hour * 3); err != nil {
		s.logger.Sugar().Errorf("failed to set prepare user username(%s): %s", createUserDto.Username, err.Error())
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

	if err := s.rabbitmq.PublishToQueue(rabbitmq.REGISTRATION_CODE_MAIL_QUEUE, queueData); err != nil {
		s.logger.Sugar().Errorf("failed to publish to rabbitmq queue(%s): %s", rabbitmq.REGISTRATION_CODE_MAIL_QUEUE, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *authService) VerifyRegistrationCodeAndCreateUser(ctx context.Context, code int) (*dto.GetUserDto, *jwtmanager.JWTPair, error) {
	// Verifying if code exists
	redisKey := redisrepo.TempRegistrationCodeKey(code)
	userData, err := redisrepo.Get[dto.CreateUserReq](s.repo.Redis.Default, ctx, redisKey)
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

	jwtPair, err := jwtmanager.GenerateJWTPair(jwtmanager.GenerateJWTPairData{
		AccessMethod: jwt.SigningMethodHS256,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": createdUser.ID.String(),
			"role": createdUser.Role,
		},
		AccessExpiry: ACCESS_TOKEN_EXPIRY,
		RefreshMethod: jwt.SigningMethodHS256,
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

	if err := s.repo.Redis.Default.Del(
		ctx,
		redisrepo.PrepareUserEmailKey(createdUser.Email),
		redisrepo.PrepareUsernameKey(createdUser.Username),
	).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete user(%s) prepare keys from redis: %s", createdUser.ID, err.Error())
	}

	user, err := s.userService.FindByUsername(ctx, nil, createdUser.Username)
	if err != nil {
		return nil, nil, ErrInternal
	}

	userCreatedBodyJSON, err := json.Marshal(map[string]string{
		"id": user.ID.String(),
		"username": user.Username,
	})
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal user(%s) created body to json: %s", user.ID.String(), err.Error())
		return nil, nil, ErrInternal
	}
	if err := s.rabbitmq.PublishExchange(rabbitmq.USERS_CREATED_EXCHANGE, userCreatedBodyJSON); err != nil {
		s.logger.Sugar().Errorf("failed to publish user(%s) created event to rabbitmq: %s", user.ID.String(), err.Error())
		return nil, nil, ErrInternal
	}

	return user, jwtPair, nil
}

func (s *authService) SendSignInCode(ctx context.Context, signInDto dto.SignInReq) error {
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
		code = newRandomCode(MIN_SIGNIN_CODE, MAX_SIGNIN_CODE)
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

	if err := s.rabbitmq.PublishToQueue(rabbitmq.SIGNIN_CODE_MAIL_QUEUE, queueData); err != nil {
		s.logger.Sugar().Errorf("failed to publish to rabbitmq queue(%s): %s", rabbitmq.SIGNIN_CODE_MAIL_QUEUE, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *authService) VerifySignInCodeAndSignIn(ctx context.Context, code int) (*dto.GetUserDto, *jwtmanager.JWTPair, error) {
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

	jwtPair, err := jwtmanager.GenerateJWTPair(jwtmanager.GenerateJWTPairData{
		AccessMethod: jwt.SigningMethodHS256,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": userData.ID.String(),
			"role": userData.Role,
		},
		AccessExpiry: ACCESS_TOKEN_EXPIRY,
		RefreshMethod: jwt.SigningMethodHS256,
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

	user, err := s.userService.FindByUsername(ctx, nil, userData.Username)
	if err != nil {
		return nil, nil, err
	}

	return user, jwtPair, nil
}

func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*jwtmanager.JWTPair, error) {
	decodedToken, err := jwtmanager.DecodeJWT(refreshToken, []byte(os.Getenv("REFRESH_SECRET")))
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

	jwtPair, err := jwtmanager.GenerateJWTPair(jwtmanager.GenerateJWTPairData{
		AccessMethod: jwt.SigningMethodHS256,
		AccessExpiry: ACCESS_TOKEN_EXPIRY,
		AccessSecret: []byte(os.Getenv("ACCESS_SECRET")),
		AccessClaims: jwt.MapClaims{
			"id": user.ID.String(),
			"role": user.Role,
			"exp": time.Hour * 3,
		},
		RefreshMethod: jwt.SigningMethodHS256,
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

func (s *authService) UpdatePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.repo.Postgres.User.FindPassword(ctx, userID)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get user(%s) from postgres: %s", userID.String(), err.Error())
		return ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidOldPassword
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 11)
	if err != nil {
		s.logger.Sugar().Errorf("failed to generate new password hash for user(%s): %s", userID.String(), err.Error())
		return ErrInternal
	}

	if err := s.repo.Postgres.User.UpdatePasswordHash(ctx, userID, string(newPasswordHash)); err != nil {
		s.logger.Sugar().Errorf("failed to update user(%s)'s password_hash: %s", userID.String(), err.Error())
		return ErrInternal
	}

	return nil
}

func (s *authService) RequestForgotPasswordCode(ctx context.Context, email string) error {
	user, err := s.repo.Postgres.User.FindByEmail(ctx, email)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get user by email(%s) from postgres: %s", email, err.Error())
		return ErrInternal
	}

	code := newRandomCode(1_000_000_000, 9_999_999_999)

	if err := s.repo.Redis.SetJSON(ctx, redisrepo.UserForgotPasswordCodeKey(code), user, time.Minute * 5); err != nil {
		s.logger.Sugar().Errorf("failed to set forgot-password code for user(%s) in redis: %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	bodyJSON, err := json.Marshal(dto.RabbitMQNotificateUserCodeDto{
		Email: user.Email,
		Code: code,
	})
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal body to send to user(%s) to json: %s", user.Email, err.Error())
		return ErrInternal
	}
	if err := s.rabbitmq.PublishToQueue(rabbitmq.USER_FORGOT_PASSWORD_QUEUE, bodyJSON); err != nil {
		s.logger.Sugar().Errorf("failed to publish message to queue(%s) for user(%s): %s", rabbitmq.USER_FORGOT_PASSWORD_QUEUE, user.ID.String(), err.Error())
		return ErrInternal
	}
	
	return nil
}

func (s *authService) ChangeForgottenPasswordByCode(ctx context.Context, req dto.ChangeForgottenPasswordReq) error {
	user, err := redisrepo.Get[model.User](s.repo.Redis.Default, ctx, redisrepo.UserForgotPasswordCodeKey(req.Code))
	if err != nil {
		if err == redis.Nil {
			return ErrInvalidForgotPasswordCode
		}

		s.logger.Sugar().Errorf("failed to get user by forgot-password code from redis: %s", err.Error())
		return ErrInternal
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 11)
	if err != nil {
		s.logger.Sugar().Errorf("failed to generate new password hash for user(%s): %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	if err := s.repo.Postgres.User.UpdatePasswordHash(ctx, user.ID, string(newPasswordHash)); err != nil {
		s.logger.Sugar().Errorf("failed to update user(%s)'s password hash: %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	if err := s.repo.Redis.Default.Del(ctx, redisrepo.UserForgotPasswordCodeKey(req.Code)).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete user(%s) forgot-password code value from redis: %s", user.ID.String(), err.Error())
	}

	return nil
}
