package postgres

import (
	"fmt"

	"github.com/bentalebwael/faceit-users-service/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/jmoiron/sqlx"
)

// NewConnection establishes a new database connection using the provided configuration
func NewConnection(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", cfg.DB.URL)
	if err != nil {
		return nil, fmt.Errorf("error connecting to the database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.DB.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return db, nil
}

func Close(db *sqlx.DB) error {
	if db != nil {
		if err := db.Close(); err != nil {
			return fmt.Errorf("error closing database connection: %w", err)
		}
	}
	return nil
}
