//go:build ignore

package main

import (
	"os"

	"github.com/hyoureii/hrbackend/internal/config"
	"github.com/hyoureii/hrbackend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const help string = `Usage:
	push		push new schema to db
	migrate		migrate new schema to db (unimplemented)
`

func main() {
	if len(os.Args) <= 1 {
		panic(help)
	}
	cmd := os.Args[1]

	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.AuthDbDsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	m := db.Migrator()
	switch cmd {
	case "push":
		tables, err := m.GetTables()
		if err != nil {
			panic(err)
		}

		if len(tables) > 0 {
			for _, table := range tables {
				err = m.DropTable(table)
				if err != nil {
					panic(err)
				}
			}
		}

		err = m.CreateTable(&models.User{})
		if err != nil {
			panic(err)
		}
		err = m.CreateTable(&models.RefreshToken{})
		if err != nil {
			panic(err)
		}
	default:
		panic(help)
	}
}
