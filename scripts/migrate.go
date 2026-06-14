//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	db, err := gorm.Open(postgres.Open(cfg.DbDsn), &gorm.Config{})
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

		createEnum("hr_request_status", goEnumToDB(models.RequestStatuses), db)
		createEnum("hr_leave_type", goEnumToDB(models.LeaveTypes), db)
		createEnum("hr_trip_type", goEnumToDB(models.TripTypes), db)
		createEnum("hr_qr_type", goEnumToDB(models.QrTypes), db)

		if err := m.CreateTable(
			&models.Role{},
			&models.Permission{},
			&models.RolePermission{},
			&models.User{},
			&models.RefreshToken{},
			&models.QrCode{},
			&models.Attendance{},
			&models.Leave{},
			&models.Trip{},
		); err != nil {
			panic(err)
		}

	default:
		panic(help)
	}
}

func createEnum(name, query string, db *gorm.DB) {
	gorm.G[any](db).Exec(
		context.Background(),
		fmt.Sprintf("DROP TYPE IF EXISTS %s", name),
	)
	if err := gorm.G[any](db).Exec(
		context.Background(),
		fmt.Sprintf("CREATE TYPE %s AS ENUM %s", name, query),
	); err != nil {
		panic(err)
	}
}

func goEnumToDB[T ~string](enum []T) string {
	dbEnum := ""
	for index, item := range enum {
		strItem := string(item)
		strItem = fmt.Sprintf("'%s'", strItem)
		if index == 0 {
			dbEnum = strItem
		} else {
			dbEnum = strings.Join([]string{dbEnum, strItem}, ",")
		}
	}
	dbEnum = fmt.Sprintf("(%s)", dbEnum)
	return dbEnum
}
