package main

import (
	"context"
	"log"
	"time"

	"github.com/hyoureii/hrbackend/internal/config"
)

func main() {
	cfg := config.Load()
	s, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	err = s.Run(context.Background(), 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to run server: %s", err)
	}
}
