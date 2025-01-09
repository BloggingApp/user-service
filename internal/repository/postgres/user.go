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
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		ctx,
		"INSERT INTO users(id, email, username, password_hash, display_name, bio, created_at, updated_at) VALUES($1, $2, $3, $4, $5, $6, $7)",
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

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.QueryRow(ctx, `
	SELECT
	u.id, u.email, u.username, u.password_hash, u.display_name, u.bio, u.created_at, u.updated_at
	FROM users u
	WHERE u.id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.QueryRow(ctx, `
	SELECT
	u.id, u.email, u.username, u.password_hash, u.display_name, u.bio, u.created_at, u.updated_at
	FROM users u
	WHERE u.email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.QueryRow(ctx, `
	SELECT
	u.id, u.email, u.username, u.password_hash, u.display_name, u.bio, u.created_at, u.updated_at
	FROM users u
	WHERE u.username = $1
	`, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Bio,
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
