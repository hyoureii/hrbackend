//go:build ignore

package main

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/config"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DbDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %s", err)
	}

	dummyUsers := []models.User{
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Hafizryandin Haykal",
			LastName:  "Matondang",
			Role:      users.Role_ROLE_ADMIN,
			Email:     "admin@hrconnect.org",
			Password:  hash("admin123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Fathir",
			LastName:  "RIH",
			Role:      users.Role_ROLE_DIRECTOR,
			Email:     "director@hrconnect.org",
			Password:  hash("director123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Nopal",
			LastName:  "Pradana",
			Role:      users.Role_ROLE_MANAGER,
			Email:     "manager@hrconnect.org",
			Password:  hash("manager123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Haidar",
			LastName:  "Zahran",
			Role:      users.Role_ROLE_SUPERVISOR,
			Email:     "supervisor@hrconnect.org",
			Password:  hash("supervisor123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Cecep",
			LastName:  "Wijaya",
			Role:      users.Role_ROLE_STAFF_UNSPECIFIED,
			Email:     "staff@hrconnect.org",
			Password:  hash("staff123"),
		},
	}

	for _, user := range dummyUsers {
		err = gorm.G[models.User](db).Create(context.Background(), &user)
		if err != nil {
			log.Fatalf("Failed to seed: %s", err)
		}
	}
}

func hash(p string) string {
	h, err := lib.HashPassword(p)
	if err != nil {
		log.Fatalf("Failed to hash password: %s", err)
	}
	return string(h)
}
