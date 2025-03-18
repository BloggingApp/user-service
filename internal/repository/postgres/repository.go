package postgres

import (
	"context"

	"github.com/BloggingApp/user-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User interface {
	Create(ctx context.Context, user model.User) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.FullUser, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.FullUser, error)
	FindByEmailOrUsername(ctx context.Context, email string, username string) (*model.User, error)
	UpdateByID(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	SearchByUsername(ctx context.Context, username string, limit int, offset int) ([]*model.FullUser, error)
	FindUserFollowers(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error)
	Follow(ctx context.Context, follower model.Follower) error
	Unfollow(ctx context.Context, follower model.Follower) error
	FindUserFollows(ctx context.Context, id uuid.UUID, limit int, offset int) ([]*model.FullFollower, error)
	ExistsWithID(ctx context.Context, id uuid.UUID) (bool, error)
	ExistsWithUsername(ctx context.Context, username string) (bool, error)
	FindUserSocialLinks(ctx context.Context, userID uuid.UUID) ([]*model.SocialLink, error)
	AddSocialLink(ctx context.Context, link model.SocialLink) error
	DeleteSocialLink(ctx context.Context, userID uuid.UUID, platform string) error
}

type PostgresRepository struct {
	User
}

func New(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		User: newUserRepo(db),
	}
}
