// Command bot runs the meet-up Telegram bot.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"meet-up-bot/internal/config"
	"meet-up-bot/internal/logger"
	"meet-up-bot/internal/storage"
	"meet-up-bot/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Env, cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	// Cancel everything on SIGINT/SIGTERM for a clean shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := storage.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	log.Info("connected to database")

	b, err := telegram.New(cfg.TelegramToken, store, log)
	if err != nil {
		return err
	}

	b.Start(ctx) // blocks until ctx is cancelled
	log.Info("shutting down")
	return nil
}
