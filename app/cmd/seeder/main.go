package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/config"
	"github.com/DiLRandI/solr-with-monitoring/app/internal/seeder/runner"
)

func main() {
	cfg, err := config.Load(os.Args[1:], os.LookupEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.SlogLevel(),
	}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runner.Run(ctx, cfg, logger); err != nil {
		logger.Error("seeder exited with error", "error", err)
		os.Exit(1)
	}
}
