package migrator

import (
	"context"
	"database/sql"
	"errors"
	"obsidian-auth/internal/seeder"
	postgresqlstorage "obsidian-auth/pkg/storage/postgresql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pgURI string, migrationsURL string, rollback int, seed bool) error {
	db, err := sql.Open("postgres", pgURI)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic(err)
	}
	defer driver.Close()

	m, err := migrate.NewWithDatabaseInstance(
		migrationsURL,
		"postgres",
		driver,
	)

	if err != nil {
		panic(err)
	}

	if rollback > 0 {
		err = m.Steps(rollback * -1)
	} else if rollback < 0 {
		err = m.Down()
	} else {
		err = m.Up()
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		panic(err)
	}

	if seed && rollback == 0 {
		pool, err := pgxpool.New(ctx, pgURI)
		if err != nil {
			panic(err)
		}
		defer pool.Close()

		u := seeder.NewUsersSeeder(postgresqlstorage.NewUserStorage(pool))
		err = u.Run(ctx)
		if err != nil {
			panic(err)
		}
	}

	return nil
}
