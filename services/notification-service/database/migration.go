package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB, dbName string) error {
	log.Printf("Starting database migrations for %s...", dbName)

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("could not create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///root/db/migrations",
		dbName,
		driver,
	)
	if err != nil {
		log.Printf("Failed to create migrate instance from /app/db/migrations, trying ./db/migrations: %v", err)
		m, err = migrate.NewWithDatabaseInstance(
			"file://db/migrations",
			dbName,
			driver,
		)
		if err != nil {
			return fmt.Errorf("could not create migrate instance: %w", err)
		}
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database migration: no changes needed")
			return nil
		}
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	log.Println("Database migration: success")
	return nil
}
