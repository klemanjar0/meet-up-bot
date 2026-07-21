// Command migrate applies (or rolls back) database migrations.
//
// Usage:
//
//	migrate up            apply all pending migrations (default)
//	migrate down          roll back the most recent migration
//	migrate drop          drop everything in the database
//	migrate force <ver>   set the migration version without running anything
//	migrate version       print the current migration version
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"meet-up-bot/db/migrations"
	"meet-up-bot/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	// The migrate tool only needs the database URL, not the bot token.
	databaseURL := config.DatabaseURL()
	if databaseURL == "" {
		return errors.New("DATABASE_URL (or POSTGRES_* vars) is required")
	}

	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "drop":
		err = m.Drop()
	case "force":
		if len(os.Args) < 3 {
			return errors.New("force requires a version argument")
		}
		v, convErr := strconv.Atoi(os.Args[2])
		if convErr != nil {
			return fmt.Errorf("invalid version %q: %w", os.Args[2], convErr)
		}
		err = m.Force(v)
	case "version":
		v, dirty, verErr := m.Version()
		if errors.Is(verErr, migrate.ErrNilVersion) {
			fmt.Println("no migrations applied")
			return nil
		}
		if verErr != nil {
			return verErr
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("no migration changes to apply")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Printf("migrate %s: ok\n", cmd)
	return nil
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}
	// The golang-migrate pgx/v5 driver registers the "pgx5" URL scheme.
	dbURL := databaseURL
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(dbURL, prefix) {
			dbURL = "pgx5://" + strings.TrimPrefix(dbURL, prefix)
			break
		}
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}
	return m, nil
}
