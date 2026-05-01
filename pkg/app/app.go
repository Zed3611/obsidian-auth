package app

import (
	"context"
	"log/slog"
	appgrpc "obsidian-auth/pkg/app/grpc"
	cacheredis "obsidian-auth/pkg/cache/redis"
	"obsidian-auth/pkg/health"
	authservice "obsidian-auth/pkg/service/auth"
	postgresqlstorage "obsidian-auth/pkg/storage/postgresql"
	"time"

	pg "github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Server        *appgrpc.App
	HealthChecker *health.Checker
}

func New(
	ctx context.Context,
	logger *slog.Logger,
	port int,
	pgConnect string,
	jwtSecret string,
	accessTokenDuration time.Duration,
	sessionDuration time.Duration,
	redisAddr string,
	redisPass string,
	redisDb int,
	healthCheckInterval time.Duration,
	healthCheckTimeout time.Duration,
) (*App, error) {
	pool, err := pg.New(ctx, pgConnect)
	if err != nil {
		return nil, err
	}

	userRepo := postgresqlstorage.NewUserStorage(pool)
	sessionRepo := postgresqlstorage.NewSessionStorage(pool)
	redis := cacheredis.New(redisAddr, redisPass, redisDb)

	healthChecker := health.New(
		logger,
		pool,
		redis,
		healthCheckInterval,
		healthCheckTimeout,
	)

	authService := authservice.New(
		authservice.AuthConfig{
			JwtSecret:           jwtSecret,
			AccessTokenDuration: accessTokenDuration,
			SessionDuration:     sessionDuration,
		},
		logger,
		redis,
		sessionRepo,
		userRepo,
	)

	app := appgrpc.New(
		logger,
		port,
		authService,
		healthChecker.Server(),
	)

	return &App{app, healthChecker}, nil
}
