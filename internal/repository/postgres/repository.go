package postgres

import (
	"context"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type User interface {
	Create(ctx context.Context, user model.User) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
}

type PostgresRepository struct {
	User
}

func New(db *pgx.Conn) *PostgresRepository {
	return &PostgresRepository{
		User: newUserRepo(db),
	}
}
