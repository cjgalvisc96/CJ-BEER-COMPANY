// Package database opens the shared Postgres pool (pgx through
// database/sql) and fails fast when the database is unreachable.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver
)

const pingTimeout = 5 * time.Second

func Open(dbURL string) (*sql.DB, error) {
	return open("pgx", dbURL)
}

func open(driverName, dbURL string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dbURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
