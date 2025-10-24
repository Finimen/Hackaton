package storage

import (
	"NetScan/internal/config"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(cfg *config.DatabaseConfig, log *slog.Logger) (*sql.DB, error) {
	connStr := cfg.GetDNS()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Error("Failed to open connection to postgres")
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	if err = db.Ping(); err != nil {
		log.Error("Failed to ping database")
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	log.Info("Successfully connected to postgres database")
	return db, nil
}
