package app

import (
	"context"
	"log/slog"
	appgrpc "obsidian-auth/pkg/app/grpc"
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
) {
	pool, _ := pg.New(ctx, pgConnect)

	userRepo := postgresqlstorage.New(pool)

	authService := authservice.New(
		authservice.AuthConfig{
			JwtSecret:           jwtSecret,
			AccessTokenDuration: accessTokenDuration,
			SessionDuration:     sessionDuration,
		},
		logger,
		userRepo,
		userRepo,
		userRepo,
	)

	_ = appgrpc.New(
		log,
		port,
		authService,
	)
}
