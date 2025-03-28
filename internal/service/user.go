package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/BloggingApp/user-service/internal/dto"
	"github.com/BloggingApp/user-service/internal/model"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/internal/repository/redisrepo"
	urlverifier "github.com/davidmytton/url-verifier"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type userService struct {
	logger *zap.Logger
	repo *repository.Repository
	rabbitmq *rabbitmq.MQConn
	httpClient *http.Client
	socialLinkTypes map[string]string
}

const (
	MIN_REGISTRATION_CODE = 100_000
	MAX_REGISTRATION_CODE = 999_999

	MIN_SIGNIN_CODE = 100_000
	MAX_SIGNIN_CODE = 999_999

	MAX_SEARCH_LIMIT = 10

	GITHUB_LINK_TYPE = "github"
	TELEGRAM_LINK_TYPE = "telegram"
	MAX_SOCIAL_LINKS_COUNT = 2
)

func newUserService(logger *zap.Logger, repo *repository.Repository, rabbitmq *rabbitmq.MQConn) User {
	return &userService{
		logger: logger,
		repo: repo,
		rabbitmq: rabbitmq,
		httpClient: &http.Client{},
		socialLinkTypes: map[string]string{
			"https://github.com/": GITHUB_LINK_TYPE,
			"https://t.me/": TELEGRAM_LINK_TYPE,
		},
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

func (s *userService) FindByUsername(ctx context.Context, getterID *uuid.UUID, username string) (*dto.GetUserDto, error) {
	userCache, err := redisrepo.Get[dto.GetUserDto](s.repo.Redis.Default, ctx, redisrepo.UserByUsernameKey(username))
	if err == nil {
		return userCache, nil
	}

	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get user from redis: %s", err.Error())
		return nil, ErrInternal
	}

	user, err := s.repo.Postgres.User.FindByUsername(ctx, *getterID, username)
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

func (s *userService) FindUserFollowers(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error) {
	maximumLimit(&limit)

	followersCache, err := redisrepo.GetMany[model.FullFollower](s.repo.Redis.Default, ctx, redisrepo.UserFollowersKey(id.String(), limit, offset))
	if err == nil {
		return followersCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get followers from redis: %s", err.Error())
		return nil, ErrInternal
	}

	followers, err := s.repo.Postgres.User.FindUserFollowers(ctx, id, limit, offset)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}

		s.logger.Sugar().Errorf("failed to get user(%s) followers from postgres: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserFollowersKey(id.String(), limit, offset), followers, time.Minute * 1); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) followers in redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	return followers, nil
}

func (s *userService) Follow(ctx context.Context, follower model.Follower) error {
	if follower.FollowerID.String() == follower.UserID.String() {
		return ErrFollowToYourself
	}

	isFollowing, err := s.repo.Redis.Default.Get(ctx, redisrepo.IsFollowingKey(follower.FollowerID.String(), follower.UserID.String())).Bool()
	if err != nil && err != redis.Nil {
		return ErrInternal
	}
	if isFollowing {
		return ErrAlreadyFollowing
	}

	defer func()  {
		if err := s.repo.Redis.Default.Set(ctx, redisrepo.IsFollowingKey(follower.FollowerID.String(), follower.UserID.String()), true, time.Minute * 5); err != nil {
			s.logger.Sugar().Errorf("failed to set isFollowed in redis: %s", err.Error())
		}
	}()

	userExists, err := s.repo.Postgres.User.ExistsWithID(ctx, follower.UserID)
	if err != nil {
		s.logger.Sugar().Errorf("failed to get exists(%s) from postgres: %s", follower.UserID, err.Error())
		return ErrInternal
	}
	if !userExists {
		return ErrUserNotFound
	}

	if err := s.repo.Postgres.User.Follow(ctx, follower); err != nil {
		s.logger.Sugar().Errorf("failed to subscribe user(%s) on user(%s) in postgres: %s", follower.FollowerID.String(), follower.UserID.String(), err.Error())
		return ErrInternal
	}

	// Sending follow to rabbitmq
	followMQBody, err := json.Marshal(map[string]interface{}{
		"user_id": follower.UserID,
		"follower_id": follower.FollowerID,
	})
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal followMQ body: %s", err.Error())
		return ErrInternal
	}
	if err := s.rabbitmq.PublishToQueue(rabbitmq.FOLLOWS_QUEUE, followMQBody); err != nil {
		s.logger.Sugar().Errorf("failed to publish follow to queue: %s", err.Error())
		return ErrInternal
	}

	// Delete cache
	if err := s.repo.Redis.Default.Del(
		ctx,
		redisrepo.UserFollowersKey(follower.UserID.String(), MAX_SEARCH_LIMIT, 0),
		redisrepo.UserFollowersKey(follower.UserID.String(), MAX_SEARCH_LIMIT, 1),
		redisrepo.UserFollowsKey(follower.FollowerID.String(), MAX_SEARCH_LIMIT, 0),
		redisrepo.UserFollowsKey(follower.FollowerID.String(), MAX_SEARCH_LIMIT, 1),
	).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete redis cache: %s", err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) Unfollow(ctx context.Context, follower model.Follower) error {
	isFollowing, err := s.repo.Redis.Get(ctx, redisrepo.IsFollowingKey(follower.FollowerID.String(), follower.UserID.String())).Bool()
	if err != nil && err != redis.Nil {
		return ErrInternal
	}
	if !isFollowing {
		return nil
	}

	defer func()  {
		if err := s.repo.Redis.Default.Set(ctx, redisrepo.IsFollowingKey(follower.FollowerID.String(), follower.UserID.String()), true, time.Minute * 5); err != nil {
			s.logger.Sugar().Errorf("failed to set isFollowed in redis: %s", err.Error())
		}
	}()

	if err := s.repo.Postgres.User.Unfollow(ctx, follower); err != nil {
		s.logger.Sugar().Errorf("failed to unfollow follower(%s) from user(%s): %s", follower.FollowerID.String(), follower.UserID.String(), err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) FindUserFollows(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error) {
	maximumLimit(&limit)

	followsCache, err := redisrepo.GetMany[model.FullFollower](s.repo.Redis.Default, ctx, redisrepo.UserFollowsKey(id.String(), limit, offset))
	if err == nil {
		return followsCache, nil
	}
	if err != redis.Nil {
		s.logger.Sugar().Errorf("failed to get follows from redis: %s", err.Error())
		return nil, ErrInternal
	}

	follows, err := s.repo.Postgres.User.FindUserFollows(ctx, id, limit, offset)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}

		s.logger.Sugar().Errorf("failed to get user(%s) follows from postgres: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	if err := s.repo.Redis.Default.SetJSON(ctx, redisrepo.UserFollowsKey(id.String(), limit, offset), follows, time.Minute * 1); err != nil {
		s.logger.Sugar().Errorf("failed to set user(%s) follows in redis: %s", id.String(), err.Error())
		return nil, ErrInternal
	}

	return follows, nil
}

func maximumLimit(limit *int) {
	if *limit > MAX_SEARCH_LIMIT {
		*limit = MAX_SEARCH_LIMIT
	}
}

func (s *userService) Update(ctx context.Context, user model.FullUser, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
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
	if err := s.deleteUserInfoCache(ctx, user); err != nil {
		return err
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

	if !filetype.IsImage(buff) {
		return ErrFileMustBeImage
	}

	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		return ErrFileMustHaveValidExtension
	}

	uploadPath := "user-avatars/" + user.ID.String()

	// Upload avatar to BloggingApp's CDN
	returnedURL, err := s.uploadAvatarToCDN(uploadPath, file, fileHeader)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"avatar_url": returnedURL,
	}
	if err := s.repo.Postgres.User.UpdateByID(ctx, user.ID, updates); err != nil {
		s.logger.Sugar().Errorf("failed to update user(%s) avatar: %s", user.ID.String(), err.Error())
		return ErrInternal
	}
	// Publish RabbitMQ event to update user info cache in other microservices
	if err := s.publishUserInfoUpdated(user.ID, updates); err != nil {
		return err
	}

	if err := s.deleteUserInfoCache(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *userService) uploadAvatarToCDN(path string, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	endpoint := "/upload"
	url := viper.GetString("cdn.origin") + endpoint

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Writing text fields
	if err := writer.WriteField("type", "IMAGE"); err != nil {
		s.logger.Sugar().Errorf("failed to write 'type' field for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	if err := writer.WriteField("path", path); err != nil {
		s.logger.Sugar().Errorf("failed to write 'path' field for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	// Writing file
	fileWriter, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create file part for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		s.logger.Sugar().Errorf("failed to seek to the start of the file: %s", err.Error())
		return "", ErrInternal
	}

	if _, err := io.Copy(fileWriter, file); err != nil {
		s.logger.Sugar().Errorf("failed to copy file content for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	// End of request body
	if err := writer.Close(); err != nil {
		s.logger.Sugar().Errorf("failed to close writer for CDN request: %s", err.Error())
		return "", ErrInternal
	}

	req, err := http.NewRequest(http.MethodPost, url, &requestBody)
	if err != nil {
		s.logger.Sugar().Errorf("failed to create CDN request: %s", err.Error())
		return "", ErrInternal
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Sugar().Errorf("failed to do CDN request: %s", err.Error())
		return "", ErrInternal
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Sugar().Errorf("failed to read response body from CDN: %s", err.Error())
		return "", ErrInternal
	}

	if resp.StatusCode != http.StatusOK {
		var bodyJSON map[string]interface{}
        if err := json.Unmarshal(body, &bodyJSON); err != nil {
            s.logger.Sugar().Errorf("failed to decode error response from CDN: %s", err.Error())
        } else {
            s.logger.Sugar().Errorf("ERROR from CDN endpoint(%s), code(%d), details: %s", endpoint, resp.StatusCode, bodyJSON["details"])
        }
        return "", ErrFailedToUploadAvatarToCDN
	}

	return string(body), nil
}

func (s *userService) publishUserInfoUpdated(userID uuid.UUID, updates map[string]interface{}) error {
	updates["user_id"] = userID.String()

	bytes, err := json.Marshal(updates)
	if err != nil {
		s.logger.Sugar().Errorf("failed to marshal user(%s) updates to json: %s", userID.String(), err.Error())
		return ErrInternal
	}
	if err := s.rabbitmq.PublishExchange(rabbitmq.USERS_UPDATE_EXCHANGE, bytes); err != nil {
		s.logger.Sugar().Errorf("failed to publish rabbitmq event to exchange(%s): %s", rabbitmq.USERS_UPDATE_EXCHANGE, err.Error())
		return ErrInternal
	}

	return nil
}

func (s *userService) AddSocialLink(ctx context.Context, user model.FullUser, link string) error {
	if len(user.SocialLinks) >= MAX_SOCIAL_LINKS_COUNT {
		return ErrMaxSocialLinksAchieved
	}

	verifier := urlverifier.NewVerifier()
	verifier.DisableHTTPCheck()
	verifier.DisallowHTTPCheckInternal()
	result, err := verifier.Verify(link)
	if err != nil {
		return err
	}

	if !result.HTTP.IsSuccess {
		return fmt.Errorf("the url is reachable with status code: %d", result.HTTP.StatusCode)
	}

	linkPlatform, err := s.defineSocialLinkType(link)
	if err != nil {
		return err
	}

	for _, l := range user.SocialLinks {
		if l.Platform == linkPlatform {
			return fmt.Errorf("link with type '%s' has already been set", l.Platform)
		}
	}

	if err := s.repo.Postgres.User.AddSocialLink(ctx, model.SocialLink{
		UserID: user.ID,
		URL: link,
		Platform: linkPlatform,
	}); err != nil {
		s.logger.Sugar().Errorf("failed to add social link for user(%s): %s", user.ID.String(), err.Error())
		return ErrInternal
	}

	if err := s.deleteUserInfoCache(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *userService) defineSocialLinkType(link string) (string, error) {
	typ := ""
	for uri, t := range s.socialLinkTypes {
		if strings.HasPrefix(link, uri) {
			typ = t
		}
	}

	if typ == "" {
		return "", ErrLinkHasInvalidType
	}

	return typ, nil
}

func (s *userService) DeleteSocialLink(ctx context.Context, user model.FullUser, platform string) error {
	if err := s.repo.Postgres.User.DeleteSocialLink(ctx, user.ID, platform); err != nil {
		s.logger.Sugar().Errorf("failed to delete user(%s) social link(%s): %s", user.ID.String(), platform, err.Error())
		return err
	}

	if err := s.deleteUserInfoCache(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *userService) deleteUserInfoCache(ctx context.Context, user model.FullUser) error {
	if err := s.repo.Redis.Default.Del(
		ctx,
		redisrepo.UserByUsernameKey(user.Username),
		redisrepo.UserKey(user.ID.String()),
	).Err(); err != nil {
		s.logger.Sugar().Errorf("failed to delete user(%s) cache: %s", user.ID.String(), err.Error())
		return ErrInternal
	}
	return nil
}
