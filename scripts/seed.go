//go:build ignore

package main

import (
	"context"
	"log"

	usersv1 "github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/config"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.AuthDbDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %s", err)
	}

	dummyUsers := []models.User{
		{
			FirstName: "Hafizryandin Haykal",
			LastName: "Matondang",
			Role: usersv1.Role_ROLE_ADMIN,
			Email: "admin@hrconnect.org",
			Password: hash("admin123"),
		},
		{
			FirstName: "Fathir",
			LastName: "RIH",
			Role: usersv1.Role_ROLE_DIRECTOR,
			Email: "director@hrconnect.org",
			Password: hash("director123"),
		},
		{
			FirstName: "Nopal",
			LastName: "Pradana",
			Role: usersv1.Role_ROLE_MANAGER,
			Email: "manager@hrconnect.org",
			Password: hash("manager123"),
		},
		{
			FirstName: "Haidar",
			LastName: "Zahran",
			Role: usersv1.Role_ROLE_SUPERVISOR,
			Email: "supervisor@hrconnect.org",
			Password: hash("supervisor123"),
		},
		{
			FirstName: "Cecep",
			LastName: "Wijaya",
			Role: usersv1.Role_ROLE_STAFF_UNSPECIFIED,
			Email: "staff@hrconnect.org",
			Password: hash("staff123"),
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
