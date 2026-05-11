package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hyoureii/hrbackend/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	s, err := NewServer(logger, cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create server: %s", err))
		return
	}

	err = s.Run(context.Background(), 10*time.Second)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to run server: %s", err))
		return
	}
}
