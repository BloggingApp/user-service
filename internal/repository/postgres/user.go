package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MAX_LIMIT = 50

type userRepo struct {
	db *pgxpool.Pool
}

func newUserRepo(db *pgxpool.Pool) User {
	return &userRepo{
		db: db,
	}
}

func (r *userRepo) Create(ctx context.Context, user model.User) (*model.User, error) {
	user.ID = uuid.New()
	user.AvatarURL = nil
	user.Role = "user"
	user.Followers = 0
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		ctx,
		"INSERT INTO users(id, email, username, password_hash, created_at, updated_at) VALUES($1, $2, $3, $4, $5, $6)",
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return &user, err
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.FullUser, error) {
	rows, err := r.db.Query(
		ctx,
		`
		SELECT
		u.id, u.email, u.username, u.password_hash, u.display_name, u.avatar_url, u.bio, u.role, u.followers, u.created_at, u.updated_at, sl.platform, sl.url
		FROM users u
		LEFT JOIN social_links sl ON u.id = sl.user_id
		WHERE u.id = $1
		`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[uuid.UUID]*model.FullUser)
	for rows.Next() {
		var (
			userID uuid.UUID
			userEmail string
			userUsername string
			userPasswordHash string
			userDisplayName *string
			userAvatarURL *string
			userBio *string
			userRole string
			userFollowers int64
			userCreatedAt time.Time
			userUpdatedAt time.Time
			socialLinkPlatform *string
			socialLinkUrl *string
		)
		if err := rows.Scan(
			&userID,
			&userEmail,
			&userUsername,
			&userPasswordHash,
			&userDisplayName,
			&userAvatarURL,
			&userBio,
			&userRole,
			&userFollowers,
			&userCreatedAt,
			&userUpdatedAt,
			&socialLinkPlatform,
			&socialLinkUrl,
		); err != nil {
			return nil, err
		}

		user, exists := userMap[userID]
		if !exists {
			user = &model.FullUser{
				ID: userID,
                Email: userEmail,
                Username: userUsername,
				PasswordHash: userPasswordHash,
                DisplayName: userDisplayName,
                AvatarURL: userAvatarURL,
                Bio: userBio,
                Role: userRole,
				Followers: userFollowers,
                CreatedAt: userCreatedAt,
                UpdatedAt: userUpdatedAt,
                SocialLinks: []*model.SocialLink{},
			}
			userMap[userID] = user
		}

		if socialLinkPlatform != nil && socialLinkUrl != nil {
			user.SocialLinks = append(user.SocialLinks, &model.SocialLink{
				UserID: userID,
				Platform: *socialLinkPlatform,
				URL: *socialLinkUrl,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var users []*model.FullUser
	for _, user := range userMap {
		users = append(users, user)
	}

	if len(users) == 0 {
		return nil, pgx.ErrNoRows
	}

	return users[0], nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.QueryRow(ctx, `
	SELECT u.id, u.email, u.username, u.password_hash, u.display_name, u.avatar_url, u.bio, u.role, u.followers, u.created_at, u.updated_at
	FROM users u
	WHERE u.email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Bio,
		&user.Role,
		&user.Followers,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*model.FullUser, error) {
	rows, err := r.db.Query(
		ctx,
		`
		SELECT
		u.id, u.email, u.username, u.display_name, u.avatar_url, u.bio, u.role, u.followers, u.created_at, u.updated_at, sl.platform, sl.url
		FROM users u
		LEFT JOIN social_links sl ON u.id = sl.user_id
		WHERE u.username = $1
		`,
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[uuid.UUID]*model.FullUser)
	for rows.Next() {
		var (
			userID uuid.UUID
			userEmail string
			userUsername string
			userDisplayName *string
			userAvatarURL *string
			userBio *string
			userRole string
			userFollowers int64
			userCreatedAt time.Time
			userUpdatedAt time.Time
			socialLinkPlatform *string
			socialLinkUrl *string
		)
		if err := rows.Scan(
			&userID,
			&userEmail,
			&userUsername,
			&userDisplayName,
			&userAvatarURL,
			&userBio,
			&userRole,
			&userFollowers,
			&userCreatedAt,
			&userUpdatedAt,
			&socialLinkPlatform,
			&socialLinkUrl,
		); err != nil {
			return nil, err
		}

		user, exists := userMap[userID]
		if !exists {
			user = &model.FullUser{
				ID: userID,
                Email: userEmail,
                Username: userUsername,
                DisplayName: userDisplayName,
                AvatarURL: userAvatarURL,
                Bio: userBio,
                Role: userRole,
				Followers: userFollowers,
                CreatedAt: userCreatedAt,
                UpdatedAt: userUpdatedAt,
                SocialLinks: []*model.SocialLink{},
			}
			userMap[userID] = user
		}

		if socialLinkPlatform != nil && socialLinkUrl != nil {
			user.SocialLinks = append(user.SocialLinks, &model.SocialLink{
				UserID: userID,
				Platform: *socialLinkPlatform,
				URL: *socialLinkUrl,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var users []*model.FullUser
	for _, user := range userMap {
		users = append(users, user)
	}

	if len(users) == 0 {
		return nil, pgx.ErrNoRows
	}

	return users[0], nil
}

func (r *userRepo) FindByEmailOrUsername(ctx context.Context, email string, username string) (*model.User, error) {
	var user model.User
	if err := r.db.QueryRow(ctx, `
	SELECT u.id, u.email, u.username, u.password_hash, u.display_name, u.avatar_url, u.bio, u.role, u.followers, u.created_at, u.updated_at
	FROM users u
	WHERE u.email = $1 OR u.username = $2
	`, email, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Bio,
		&user.Role,
		&user.Followers,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) UpdateByID(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	allowedFields := []string{"username", "display_name", "bio", "avatar_url", "followers"}
	allowedFieldsSet := make(map[string]struct{}, len(allowedFields))
	for _, field := range allowedFields {
		allowedFieldsSet[field] = struct{}{}
	}

	for field := range updates {
		if _, ok := allowedFieldsSet[field]; !ok {
			delete(updates, field)
		}
	}

	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE users SET "
	args := []interface{}{}
	i := 1

	for column, value := range updates {
		query += (column + " = $" + strconv.Itoa(i) + ", ")
		args = append(args, value)
		i++
	}

	query = query[:len(query)-2] + " WHERE id = $" + strconv.Itoa(i)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *userRepo) SearchByUsername(ctx context.Context, username string, limit int, offset int) ([]*model.FullUser, error) {
	maximumLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`
		SELECT
		u.id, u.email, u.username, u.display_name, u.avatar_url, u.bio, u.role, u.followers, u.created_at, u.updated_at, sl.platform, sl.url
		FROM users u
		LEFT JOIN social_links sl ON u.id = sl.user_id
		WHERE u.username LIKE %$1%
		LIMIT $2
		OFFSET $3
		`,
		username,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[uuid.UUID]*model.FullUser)
	for rows.Next() {
		var (
			userID uuid.UUID
			userEmail string
			userUsername string
			userDisplayName *string
			userAvatarURL *string
			userBio *string
			userRole string
			userFollowers int64
			userCreatedAt time.Time
			userUpdatedAt time.Time
			socialLinkPlatform *string
			socialLinkUrl *string
		)
		if err := rows.Scan(
			&userID,
			&userEmail,
			&userUsername,
			&userDisplayName,
			&userAvatarURL,
			&userBio,
			&userRole,
			&userFollowers,
			&userCreatedAt,
			&userUpdatedAt,
			&socialLinkPlatform,
			&socialLinkUrl,
		); err != nil {
			return nil, err
		}

		user, exists := userMap[userID]
		if !exists {
			user = &model.FullUser{
				ID: userID,
                Email: userEmail,
                Username: userUsername,
                DisplayName: userDisplayName,
                AvatarURL: userAvatarURL,
                Bio: userBio,
                Role: userRole,
				Followers: userFollowers,
                CreatedAt: userCreatedAt,
                UpdatedAt: userUpdatedAt,
                SocialLinks: []*model.SocialLink{},
			}
			userMap[userID] = user
		}

		if socialLinkPlatform != nil && socialLinkUrl != nil {
			user.SocialLinks = append(user.SocialLinks, &model.SocialLink{
				UserID: user.ID,
				Platform: *socialLinkPlatform,
				URL: *socialLinkUrl,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var users []*model.FullUser
	for _, user := range userMap {
		users = append(users, user)
	}

	return users, nil
}

func maximumLimit(l *int) {
	if *l > MAX_LIMIT {
		*l = MAX_LIMIT
	}
}

func (r *userRepo) FindUserFollowers(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error) {
	maximumLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`
		SELECT f.follower_id, u.username, u.display_name, u.avatar_url, u.bio
		FROM followers f
		JOIN users u ON f.follower_id = u.id
		WHERE f.user_id = $1
		LIMIT $2
		OFFSET $3
		`,
		id,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followers []*model.FullFollower
	for rows.Next() {
		var follower model.FullFollower
		if err := rows.Scan(
			&follower.ID,
			&follower.Username,
			&follower.DisplayName,
			&follower.AvatarHash,
			&follower.Bio,
		); err != nil {
			return nil, err
		}

		followers = append(followers, &follower)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return followers, nil
}

func (r *userRepo) Follow(ctx context.Context, follower model.Follower) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(
		ctx,
		`
		SELECT EXISTS(SELECT 1 FROM followers WHERE user_id = $1 AND follower_id = $2)
		`,
		follower.UserID, follower.FollowerID,
	).Scan(&exists); err != nil {
		return err
	}

	if exists {
		return nil
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO followers(user_id, follower_id)
		VALUES($1, $2)
		ON CONFLICT DO NOTHING
		`,
		follower.UserID,
		follower.FollowerID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE users SET followers = followers + 1 WHERE id = $1", follower.UserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *userRepo) Unfollow(ctx context.Context, follower model.Follower) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(
		ctx,
		`
		SELECT EXISTS(SELECT 1 FROM followers WHERE user_id = $1 AND follower_id = $2)
		`,
		follower.UserID, follower.FollowerID,
	).Scan(&exists); err != nil {
		return err
	}

	if !exists {
		return nil
	}

	_, err = tx.Exec(
		ctx,
		`
		DELETE FROM followers
		WHERE user_id = $1 AND follower_id = $2
		`,
		follower.UserID,
		follower.FollowerID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE users SET followers = followers - 1 WHERE id = $1", follower.UserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *userRepo) IncrFollowers(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET followers = followers + 1 WHERE id = $1", userID)
	return err
}

func (r *userRepo) DecrFollowers(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET followers = followers - 1 WHERE id = $1", userID)
	return err
}

func (r *userRepo) FindUserFollows(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error) {
	maximumLimit(&limit)

	rows, err := r.db.Query(
		ctx,
		`
		SELECT f.follower_id, u.username, u.display_name, u.avatar_url, u.bio
		FROM followers f
		JOIN users u ON f.user_id = u.id
		WHERE f.follower_id = $1
		LIMIT $2
		OFFSET $3
		`,
		id,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followers []*model.FullFollower
	for rows.Next() {
		var follower model.FullFollower
		if err := rows.Scan(
			&follower.ID,
			&follower.Username,
			&follower.DisplayName,
			&follower.AvatarHash,
			&follower.Bio,
		); err != nil {
			return nil, err
		}

		followers = append(followers, &follower)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return followers, nil
}

func (r *userRepo) ExistsWithID(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users u WHERE u.id = $1)", id).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *userRepo) ExistsWithUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users u WHERE u.username = $1)", username).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *userRepo) FindUserSocialLinks(ctx context.Context, userID uuid.UUID) ([]*model.SocialLink, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT
		l.user_id, l.url, l.platform
		FROM social_links l
		WHERE l.user_id = $1
		`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*model.SocialLink
	for rows.Next() {
		var link model.SocialLink
		if err := rows.Scan(
			&link.UserID,
			&link.URL,
			&link.Platform,
		); err != nil {
			return nil, err
		}

		links = append(links, &link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func (r *userRepo) AddSocialLink(ctx context.Context, link model.SocialLink) error {
	_, err := r.db.Exec(ctx, "INSERT INTO social_links(user_id, url, platform) VALUES($1, $2, $3)", link.UserID, link.URL, link.Platform)
	return err
}

func (r *userRepo) DeleteSocialLink(ctx context.Context, userID uuid.UUID, platform string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM social_links l WHERE l.user_id = $1 AND l.platform = $2", userID, platform)
	return err
}
