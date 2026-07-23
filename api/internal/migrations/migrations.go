// Package migrations embeds SQL migrations into the binary so the same image
// that serves traffic can run `api migrate` as a one-off task.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var files embed.FS

func Up(databaseURL string) error {
	// migrate picks its database driver from the URL scheme; the pgx/v5
	// driver registers as "pgx5", while the app-facing URL uses postgres://.
	databaseURL = strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
	databaseURL = strings.Replace(databaseURL, "postgresql://", "pgx5://", 1)

	src, err := iofs.New(files, "sql")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
