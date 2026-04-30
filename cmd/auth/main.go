package main

import (
	"log/slog"
	appgrpc "obsidian-auth/pkg/app/grpc"
	authservice "obsidian-auth/pkg/service/auth"
	postgresqlstorage "obsidian-auth/pkg/storage/postgresql"
	"os"
	"strconv"

	pg "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(&slog.JSONHandler{})

	pool = pg.New()

	userRepo = postgresqlstorage.New()

	authService := authservice.New(
		authservice.AuthConfig{},
		logger,
	)

	app := appgrpc.New(
		&logger,
		getIntEnvOrDefault("GRPC_PORT", 8080),
	)
}

func getEnvOrDefault(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getIntEnvOrDefault(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		} else {
			return defaultVal
		}
	}

	return defaultVal
}
