package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userRepo struct {
	db *pgx.Conn
}

func newUserRepo(db *pgx.Conn) User {
	return &userRepo{
		db: db,
	}
}

func (r *userRepo) Create(ctx context.Context, user model.User) (*model.User, error) {
	user.ID = uuid.New()
	user.Role = "user"
	user.Subscribers = 0
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		ctx,
		"INSERT INTO users(id, email, username, password_hash, display_name, bio, created_at, updated_at) VALUES($1, $2, $3, $4, $5, $6, $7, $8)",
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.DisplayName,
		user.Bio,
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
		u.id, u.email, u.username, u.password_hash, u.display_name, u.avatar_hash, u.bio, u.role, u.subscribers, u.created_at, u.updated_at, sl.platform, sl.url
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
			userAvatarHash *string
			userBio *string
			userRole string
			userSubscribers int64
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
			&userAvatarHash,
			&userBio,
			&userRole,
			&userSubscribers,
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
                AvatarHash: userAvatarHash,
                Bio: userBio,
                Role: userRole,
				Subscribers: userSubscribers,
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
				Url: *socialLinkUrl,
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
	SELECT u.id, u.email, u.username, u.password_hash, u.display_name, u.avatar_hash, u.bio, u.role, u.subscribers, u.created_at, u.updated_at
	FROM users u
	WHERE u.email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarHash,
		&user.Bio,
		&user.Role,
		&user.Subscribers,
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
		u.id, u.email, u.username, u.display_name, u.avatar_hash, u.bio, u.role, u.subscribers, u.created_at, u.updated_at, sl.platform, sl.url
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
			userAvatarHash *string
			userBio *string
			userRole string
			userSubscribers int64
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
			&userAvatarHash,
			&userBio,
			&userRole,
			&userSubscribers,
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
                AvatarHash: userAvatarHash,
                Bio: userBio,
                Role: userRole,
				Subscribers: userSubscribers,
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
				Url: *socialLinkUrl,
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
	SELECT u.id, u.email, u.username, u.password_hash, u.display_name, u.avatar_hash, u.bio, u.role, u.subscribers, u.created_at, u.updated_at
	FROM users u
	WHERE u.email = $1 OR u.username = $2
	`, email, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarHash,
		&user.Bio,
		&user.Role,
		&user.Subscribers,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) UpdateByID(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
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
	maxLimit := 100

	if limit > maxLimit {
		limit = maxLimit
	}

	rows, err := r.db.Query(
		ctx,
		`
		SELECT
		u.id, u.email, u.username, u.display_name, u.avatar_hash, u.bio, u.role, u.subscribers, u.created_at, u.updated_at, sl.platform, sl.url
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
			userAvatarHash *string
			userBio *string
			userRole string
			userSubscribers int64
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
			&userAvatarHash,
			&userBio,
			&userRole,
			&userSubscribers,
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
                AvatarHash: userAvatarHash,
                Bio: userBio,
                Role: userRole,
				Subscribers: userSubscribers,
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
				Url: *socialLinkUrl,
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
