package main

import (
	"context"
	"log/slog"
	"obsidian-auth/pkg/app"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	logger := setupLogger(getEnvOrDefault("APP_ENV", envLocal))
	logger.Info("Test")
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

	go func() {
		a.Server.MustRun()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	a.Server.Stop()

	logger.Info("App stopped gracefully")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}),
		)
	}

	return log
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
