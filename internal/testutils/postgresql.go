package testutils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const pgUser = "test"
const pgPass = "pass"

var pgHost string
var pgPort string

func ProvideDB(t *testing.T) (string, *pgxpool.Pool) {
	t.Helper()

	if pgHost == "" && pgPort == "" {
		host, port, err := startContainer()
		if err != nil {
			t.Fatalf("Failed to start pgsql container: %v", err)
		}

		pgHost, pgPort = host, port
	}

	dbName := "test_" + uuid.New().String()[:8]

	adminURI := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", pgUser, pgPass, pgHost, pgPort)
	adminPool, err := pgxpool.New(context.Background(), adminURI)
	if err != nil {
		t.Fatalf("Failed to construct admin pgxpool: %v", err)
	}
	defer adminPool.Close()

	ctx, cancelCreate := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelCreate()
	if _, err = adminPool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	pgURI := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, dbName)
	pool, err := pgxpool.New(context.Background(), pgURI)
	if err != nil {
		t.Fatalf("Failed to construct pgxpool: %v", err)
	}

	ctx2, cancelPing := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelPing()
	if err = pool.Ping(ctx2); err != nil {
		t.Fatalf("Failed to ping pgxpool connection: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()

		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()

		dropPool, err := pgxpool.New(context.Background(), adminURI)
		if err != nil {
			t.Fatalf("Failed to construct admin pool for cleanup: %v", err)
		}
		defer dropPool.Close()

		if _, err = dropPool.Exec(dropCtx, fmt.Sprintf(`DROP DATABASE %s`, dbName)); err != nil {
			t.Fatalf("Failed to drop testing database: %v", err)
		}
	})

	return pgURI, pool
}

func startContainer() (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Name:         "testcontainers-postgresql",
			Image:        "postgres:18-alpine3.23",
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
			Env: map[string]string{
				"POSTGRES_USER":     pgUser,
				"POSTGRES_PASSWORD": pgPass,
			},
		},
		Started: true,
		Reuse:   true,
	})

	if err != nil {
		return "", "", err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", "", err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", "", err
	}

	return host, port.Port(), nil
}
