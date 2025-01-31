package service

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/internal/repository/redisrepo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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

	MAX_SEARCH_LIMIT = 10
)

func newUserService(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) User {
	return &userService{
		logger: logger,
		repo: repo,
		rabbitmq: rabbitmq,
	}
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
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}

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
	maximumLimit(&limit)

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

func (s *userService) FindUserSubscribers(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullSub, error) {
	maximumLimit(&limit)

	subsCache, err := redisrepo.GetMany[model.FullSub](s.repo.Redis.Default, ctx, redisrepo.UserSubscribersKey(id.String(), limit, offset))
	if err == nil {
		return subsCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get subscribers from redis: %s", err.Error())
		return nil, ErrInternal
	}

	subs, err := s.repo.Postgres.User.FindUserSubscribers(ctx, id, limit, offset)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}

		s.logger.Sugar().Errorf("failed to get user(%s) subscribers from postgres: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserSubscribersKey(id.String(), limit, offset), subs, time.Minute * 1); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) subscribers in redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	return subs, nil
}

func (s *userService) Subscribe(ctx context.Context, subscriber model.Subscriber) error {
	if subscriber.SubID.String() == subscriber.UserID.String() {
		return ErrSubToYourself
	}

	isSubscribed, err := s.repo.Redis.Default.Get(ctx, redisrepo.IsSubscribedKey(subscriber.SubID.String(), subscriber.UserID.String())).Bool()
	if err != nil && err != redis.Nil {
		return ErrInternal
	}
	if isSubscribed {
		return ErrCooldown
	}

	defer func()  {
		if err := s.repo.Redis.Default.Set(ctx, redisrepo.IsSubscribedKey(subscriber.SubID.String(), subscriber.UserID.String()), true, time.Minute * 5); err != nil {
			s.logger.Sugar().Errorf("failed to set isSubscribed in redis: %s", err.Error())
		}
	}()

	userExists, err := s.repo.Postgres.User.ExistsWithID(ctx, subscriber.UserID)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get exists(%s) from postgres: %s", subscriber.UserID, err.Error())
		return ErrInternal
	}
	if !userExists {
		return ErrUserNotFound
	}

	if err := s.repo.Postgres.User.Subscribe(ctx, subscriber); err != nil {
		if pgError, ok := err.(*pgconn.PgError); ok && pgError.Code == "23505" {
			return ErrAlreadySubscribed
		}

		s.logger.Sugar().Errorf("failed to subscribe user(%s) on user(%s) in postgres: %s", subscriber.SubID.String(), subscriber.UserID.String(), err.Error())
		return ErrInternal
	}

	updates := map[string]interface{}{
		"subscribers": +1,
	}
	if err := s.repo.Postgres.User.UpdateByID(ctx, subscriber.UserID, updates); err != nil {
		s.logger.Sugar().Errorf("failed to update user(%s): %s", subscriber.UserID, err.Error())
		return ErrInternal
	}

	if err := s.repo.Redis.Default.Del(
		ctx,
		redisrepo.UserSubscribersKey(subscriber.UserID.String(), MAX_SEARCH_LIMIT, 0),
		redisrepo.UserSubscribersKey(subscriber.UserID.String(), MAX_SEARCH_LIMIT, 1),
		redisrepo.UserSubscriptionsKey(subscriber.SubID.String(), MAX_SEARCH_LIMIT, 0),
		redisrepo.UserSubscriptionsKey(subscriber.SubID.String(), MAX_SEARCH_LIMIT, 1),
	).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete redis cache: %s", err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) FindUserSubscriptions(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullSub, error) {
	maximumLimit(&limit)

	subsCache, err := redisrepo.GetMany[model.FullSub](s.repo.Redis.Default, ctx, redisrepo.UserSubscriptionsKey(id.String(), limit, offset))
	if err == nil {
		return subsCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get subscriptions from redis: %s", err.Error())
		return nil, ErrInternal
	}

	subs, err := s.repo.Postgres.User.FindUserSubscriptions(ctx, id, limit, offset)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}

		s.logger.Sugar().Errorf("failed to get user(%s) subscriptions from postgres: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserSubscriptionsKey(id.String(), limit, offset), subs, time.Minute * 1); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) subscriptions in redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	return subs, nil
}

func maximumLimit(limit *int) {
	if *limit > MAX_SEARCH_LIMIT {
		*limit = MAX_SEARCH_LIMIT
	}
}

func (s *userService) Update(ctx context.Context, user model.FullUser, updates map[string]interface{}) error {
	allowedFields := []string{"username", "display_name", "bio"}
	allowedFieldsSet := make(map[string]struct{}, len(allowedFields))
	for _, field := range allowedFields {
		allowedFieldsSet[field] = struct{}{}
	}

	for field := range updates {
		if _, ok := allowedFieldsSet[field]; !ok {
			return ErrFieldsNotAllowedToUpdate
		}
	}

	if username, ok := updates["username"]; ok {
		exists, err := s.repo.Postgres.User.ExistsWithUsername(ctx, username.(string))
		if err != nil {
			s.logger.Sugar().Errorf("failed to get exists with username(%s) result from postgres: %s", username.(string), err.Error())
			return ErrInternal
		}
		if exists {
			return ErrUserWithUsernameAlreadyExists
		}
	}

	// Publish RabbitMQ event to update user info cache in other microservices
	if err := s.publishUserInfoUpdated(user.ID, updates); err != nil {
		return err
	}

	// Clear cache
	if err := s.repo.Redis.Del(
		ctx,
		redisrepo.UserKey(user.ID.String()),
		redisrepo.UserByUsernameKey(user.Username),
	).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to get user(%s) from redis: %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	if err := s.repo.Postgres.User.UpdateByID(ctx, user.ID, updates); err != nil {
		s.logger.Sugar().Errorf("failed to update user(%s): %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) SetAvatar(ctx context.Context, user model.FullUser, fileHeader *multipart.FileHeader) error {
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		return err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
        return err
    }

	if !strings.HasPrefix(http.DetectContentType(buff), "image/") {
        return ErrFileMustBeImage
    }

	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		return ErrFileMustHaveValidExtension
	}

	uploadPath := "public/user-avatars/"
	filePath := uploadPath + user.ID.String() + ext

	// Remove user previous avatar
	files, err := filepath.Glob(filepath.Join(uploadPath, user.ID.String() + ".*"))
	if err != nil {
		s.logger.Sugar().Errorf("failed to glob files: %s", err.Error())
		return ErrInternal
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			s.logger.Sugar().Errorf("failed to remove user(%s) previous avatar file: %s", user.ID.String(), err.Error())
			return ErrInternal
		}
	}

	// Create new avatar
	out, err := os.Create(filePath)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create user(%s) avatar file: %s", user.ID, err.Error())
		return ErrInternal
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		s.logger.Sugar().Errorf("failed to copy user(%s) avatar file: %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	updates := map[string]interface{}{
		"avatar_hash": "/" + filePath,
	}
	if err := s.repo.Postgres.User.UpdateByID(ctx, user.ID, updates); err != nil {
		s.logger.Sugar().Errorf("failed to update user(%s) avatar: %s", user.ID.String(), err.Error())
		return ErrInternal
	}
	// Publish RabbitMQ event to update user info cache in other microservices
	if err := s.publishUserInfoUpdated(user.ID, updates); err != nil {
		return err
	}

	if err := s.repo.Redis.Default.Del(
		ctx,
		redisrepo.UserKey(user.ID.String()),
		redisrepo.UserByUsernameKey(user.Username),
	).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to clear cache after user(%s) avatar update: %s", user.ID.String(), err.Error())
	}

	return nil
}

func (s *userService) publishUserInfoUpdated(userID uuid.UUID, updates map[string]interface{}) error {
	updates["user_id"] = userID.String()

	bytes, err := json.Marshal(updates)
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal user(%s) updates to json: %s", userID.String(), err.Error())
		return ErrInternal
	}
	if err := s.rabbitmq.Publish(rabbitmq.USER_INFO_UPDATED_QUEUE, bytes); err != nil {
		s.logger.Sugar().Errorf("failed to publish rabbitmq event to queue(%s): %s", rabbitmq.USER_INFO_UPDATED_QUEUE, err.Error())
		return ErrInternal
	}

	return nil
}
