// Command api serves the CJ Beer Company HTTP API and its in-process
// message workers.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cjgalvisc96/cj-beer-company/internal/app"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(config.Load())
	if err != nil {
		return err
	}
	return application.Run(ctx)
}
