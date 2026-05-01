package main

import (
	"context"
	"log/slog"
	"obsidian-auth/pkg/app"
	"os"
	"strconv"
	"time"
)

func main() {
	logger := slog.New(&slog.JSONHandler{})
	ctx := context.Background()

	accessTokenDuration := time.Duration(getIntEnvOrDefault("ACCESS_TOKEN_DURATION_MINUTES", 5)) * time.Minute // 5 mins
	sessionDuration := time.Duration(getIntEnvOrDefault("SESSION_DURATION_MINUTES", 10080)) * time.Minute      // 7 days

	a := app.New(
		ctx,
		logger,
		getIntEnvOrDefault("GRPC_PORT", 8080),
		getEnvOrDefault("PG_CONNECT_STRING", "postgresql://root@localhost:5432/auth"),
		getEnvOrDefault("JWT_SECRET", "test-secret-change-me"),
		accessTokenDuration,
		sessionDuration,
		getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		getEnvOrDefault("REDIS_PASS", ""),
		0,
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
