package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"obsidian-auth/internal/migrator"
)

var (
	Seed     = flag.Bool("seed", false, "If seeder should run.")
	Rollback = flag.Int("rollback", 0, "Amount of migrations to rollback.")
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	flag.Parse()

	pgURI := mustEnv("PG_CONNECT_STRING")

	migrator.Migrate(ctx, pgURI, "file://migrations", int(*Rollback), *Seed)
}

func mustEnv(param string) string {
	if res, ok := os.LookupEnv("PG_CONNECT_STRING"); ok {
		return res
	}

	panic(fmt.Sprintf("Env param %s not found", param))
}
