package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/config"
)

var databases = map[string]func(config.Config) (string, string){
	"business": func(cfg config.Config) (string, string) {
		return cfg.DatabaseURL, "file://migrations/business"
	},
	"rag": func(cfg config.Config) (string, string) {
		return cfg.RAGDatabaseURL, "file://migrations/rag"
	},
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: migrate <database> [up|down|version]\n  databases: business, rag")
	}

	dbName := os.Args[1]
	resolve, ok := databases[dbName]
	if !ok {
		log.Fatalf("unknown database %q (available: business, rag)", dbName)
	}

	cmd := "up"
	if len(os.Args) > 2 {
		cmd = os.Args[2]
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbURL, migrationsDir := resolve(cfg)

	m, err := migrate.New(migrationsDir, dbURL)
	if err != nil {
		log.Fatalf("failed to create %s migrator: %v", dbName, err)
	}
	defer func() { _, _ = m.Close() }()

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("%s migration up failed: %v", dbName, err)
		}
		fmt.Printf("%s migrations applied successfully.\n", dbName)
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("%s migration down failed: %v", dbName, err)
		}
		fmt.Printf("%s migrations rolled back successfully.\n", dbName)
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("failed to get %s version: %v", dbName, err)
		}
		fmt.Printf("%s — Version: %d, Dirty: %v\n", dbName, version, dirty)
	default:
		log.Fatalf("unknown command: %s (use: up, down, version)", cmd)
	}
}
