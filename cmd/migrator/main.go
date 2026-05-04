package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"obsidian-auth/internal/seeder"
	postgresqlstorage "obsidian-auth/pkg/storage/postgresql"

	pg "github.com/jackc/pgx/v5/pgxpool"
)

var (
	Seed     = flag.Bool("seed", false, "If seeder should run.")
	Rollback = flag.Uint("rollback", 0, "Amount of migrations to rollback.")
)

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	flag.Parse()

	pgURI := mustEnv("PG_CONNECT_STRING")

	db, err := sql.Open("postgres", pgURI)
	if err != nil {
		panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)

	if err != nil {
		panic(err)
	}

	if *Rollback != 0 {
		err = m.Steps(int(*Rollback) * -1)
	} else {
		err = m.Up()
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		panic(err)
	}

	if *Seed && *Rollback == 0 {
		pool, err := pg.New(ctx, pgURI)
		if err != nil {
			panic(err)
		}

		u := seeder.NewUsersSeeder(postgresqlstorage.NewUserStorage(pool))
		err = u.Run(ctx)
		if err != nil {
			panic(err)
		}
	}
}

func mustEnv(param string) string {
	if res, ok := os.LookupEnv("PG_CONNECT_STRING"); ok {
		return res
	}

	panic(fmt.Sprintf("Env param %s not found", param))
}
