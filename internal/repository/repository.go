package repository

import (
	"github.com/BloggingApp/user-service/internal/repository/postgres"
	"github.com/BloggingApp/user-service/internal/repository/redisrepo"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	Postgres *postgres.PostgresRepository
	Redis    *redisrepo.RedisRepository
}

func New(db *pgxpool.Pool, rdb *redis.Client) *Repository {
	return &Repository{
		Postgres: postgres.New(db),
		Redis: redisrepo.New(rdb),
	}
}
