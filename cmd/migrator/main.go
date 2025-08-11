package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/Fancu1/phoenix-rss/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("migrator error: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbURL := buildPostgresURL(cfg)

	// Fixed migrations dir relative to repo root
	absDir, err := filepath.Abs("db/migrations")
	if err != nil {
		return fmt.Errorf("resolve migrations dir: %w", err)
	}
	sourceURL := fmt.Sprintf("file://%s", absDir)

	if len(os.Args) < 2 {
		usage()
		return errors.New("no command provided")
	}

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "up":
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no change")
			return nil
		}
		return err
	case "down":
		err = m.Down()
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no change")
			return nil
		}
		return err
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil {
			if errors.Is(verr, migrate.ErrNilVersion) {
				fmt.Println("version: nil")
				return nil
			}
			return fmt.Errorf("get version: %w", verr)
		}
		fmt.Printf("version: %d dirty=%v\n", v, dirty)
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func buildPostgresURL(cfg *config.Config) string {
	// prefer environment variable DATABASE_URL if provided
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	db := cfg.Database
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		db.User, db.Password, db.Host, db.Port, db.Database, db.SSLMode,
	)
}

func usage() {
	fmt.Println("usage: migrator <command>")
	fmt.Println("commands:")
	fmt.Println("  up         apply all pending migrations")
	fmt.Println("  down       rollback all migrations")
	fmt.Println("  version    print current version")
}
