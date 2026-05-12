package migrator_test

import (
	"context"
	"obsidian-auth/internal/migrator"
	"obsidian-auth/internal/testutils"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestMigratorUpDown(t *testing.T) {
	testutils.SkipShort(t)

	t.Parallel()

	pgURI, _ := testutils.ProvideDB(t)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	_, thisFile, _, _ := runtime.Caller(0)
	migrationsURL := "file://" + filepath.ToSlash(filepath.Join(filepath.Dir(thisFile), "..", "migrations"))

	migrator.Migrate(ctx, pgURI, migrationsURL, 0, true)   // Up all
	migrator.Migrate(ctx, pgURI, migrationsURL, -1, false) // Down all

	migrator.Migrate(ctx, pgURI, migrationsURL, 0, true)   // Up all
	migrator.Migrate(ctx, pgURI, migrationsURL, -1, false) // Down all
}
