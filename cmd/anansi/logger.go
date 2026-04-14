package main

import (
	"log/slog"
	"os"
)

func setupLogger(cfg *AnansiConfig) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: cfg.SlogLevel(),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
