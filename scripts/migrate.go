//go:build ignore

package main

import (
	"log"
	"os"

	"github.com/hyoureii/hrbackend/internal/config"
	"github.com/hyoureii/hrbackend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const help string = `Args:
	push		push new schema to db
	migrate		migrate new schema to db (unimplemented)
`

func main() {
	if len(os.Args) <= 1 {
		print(help)
		return
	}
	cmd := os.Args[1]

	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.AuthDbDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %s", err)
	}

	m := db.Migrator()
	switch cmd {
	case "push":
		tables, err := m.GetTables()
		if err != nil {
			log.Fatalf("failed to get all tables: %s", err)
		}

		if len(tables) > 0 {
			for _, table := range tables {
				err = m.DropTable(table)
				if err != nil {
					log.Fatalf("failed to drop tables: %s", err)
				}
			}
		}

		err = m.CreateTable(&models.User{})
		if err != nil {
			log.Fatalf("failed to create tables: %s", err)
		}
		err = m.CreateTable(&models.RefreshToken{})
		if err != nil {
			log.Fatalf("failed to create tables: %s", err)
		}
	default:
		print(help)
	}
}
