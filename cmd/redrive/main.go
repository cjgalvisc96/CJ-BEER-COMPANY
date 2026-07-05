// Command redrive republishes archived dead letters to their original
// topics after the underlying cause is fixed — parked messages are never
// lost (book Ch. 12). Requires the durable stack: DB_URL for the archive
// and BROKER_URL so the redriven messages reach the running consumers.
//
//	DB_URL=... BROKER_URL=... go run ./cmd/redrive   (or: task redrive)
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/database"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "redrive failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if cfg.DBURL == "" || cfg.BrokerURL == "" {
		return fmt.Errorf("redrive needs DB_URL (the archive) and BROKER_URL (the shared broker)")
	}
	logger := logging.New(cfg.LogLevel)

	db, err := database.Open(cfg.DBURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	transport, err := muflone.NewAMQPTransport(cfg.BrokerURL, logger)
	if err != nil {
		return err
	}
	defer func() { _ = transport.Close() }()

	redriven, err := muflone.RedriveDeadLetters(context.Background(), db, transport.Publisher(), logger)
	if err != nil {
		return err
	}
	logger.Info("redrive.completed", "messages", redriven)
	return nil
}
