package main

import (
	"log"

	"github.com/hyoureii/hrbackend/internal/config"
)

func main() {
	cfg := config.Load()
	s, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	s.Run()
}
