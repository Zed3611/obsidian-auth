package app

import (
	"context"
	"log/slog"
	appgrpc "obsidian-auth/pkg/app/grpc"
	cacheredis "obsidian-auth/pkg/cache/redis"
	authservice "obsidian-auth/pkg/service/auth"
	postgresqlstorage "obsidian-auth/pkg/storage/postgresql"
	"time"

	pg "github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Server *appgrpc.App
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
) *App {
	pool, _ := pg.New(ctx, pgConnect)

	userRepo := postgresqlstorage.NewUserStorage(pool)
	sessionRepo := postgresqlstorage.NewSessionStorage(pool)
	redis := cacheredis.New(redisAddr, redisPass, redisDb)

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
	)

	return &App{app}
}
