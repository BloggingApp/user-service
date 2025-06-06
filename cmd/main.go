package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BloggingApp/user-service/internal/config"
	"github.com/BloggingApp/user-service/internal/handler"
	"github.com/BloggingApp/user-service/internal/rabbitmq"
	"github.com/BloggingApp/user-service/internal/repository"
	"github.com/BloggingApp/user-service/internal/repository/postgres"
	"github.com/BloggingApp/user-service/internal/server"
	"github.com/BloggingApp/user-service/internal/service"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	logger, _ := zap.NewProduction()

	if err := initEnv(); err != nil {
		logger.Sugar().Fatalf("failed to load environment variables: %s", err.Error())
	}

	if err := initConfig(); err != nil {
		logger.Sugar().Fatalf("failed to initialize yaml config: %s", err.Error())
	}

	dbConfig := config.DBConfig{
		Username: os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Host: os.Getenv("POSTGRES_HOST"),
		Port: os.Getenv("POSTGRES_PORT"),
		DBName: os.Getenv("POSTGRES_DATABASE"),
		SSLMode: os.Getenv("POSTGRES_SSLMODE"),
	}
	db, err := postgres.NewDB(ctx, dbConfig)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %s", err.Error())
	}
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping postgres: %s", err.Error())
	}
	log.Println("Successfully connected to PostgreSQL")

	redisOptions := &redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	}
	rdb := redis.NewClient(redisOptions)
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("failed to ping redis: %s", err.Error())
	}
	log.Printf("Successfully connected to Redis: %s", pong)

	rabbitmq, err := rabbitmq.New(os.Getenv("RABBITMQ_CONN_STRING"))
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %s", err.Error())
	}
	log.Println("Successfully connected to RabbitMQ")

	repos := repository.New(db, rdb)
	services := service.New(logger, repos, rabbitmq)
	handlers := handler.New(services)

	srv := server.New()
	serverConfig := config.ServerConfig{
		Port: viper.GetString("app.port"),
		Handler: handlers.InitRoutes(),
		MaxHeaderBytes: 1 << 20,
		ReadTimeout: time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	go func(srv server.Server, cfg config.ServerConfig) {
		if err := srv.Run(cfg); err != nil {
			log.Fatalf("failed to run http server: %s", err.Error())
		}
	}(*srv, serverConfig)

	log.Println("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("Server shutting Down")
}

func initEnv() error {
	return godotenv.Load(".env")
}

func initConfig() error {
	viper.AddConfigPath("./")
	viper.SetConfigType("yaml")
	viper.SetConfigName("app")
	return viper.ReadInConfig()
}
