//go:build ignore

package main

import (
	"context"
	"log"

	"github.com/google/uuid"
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

	roles := []models.Role{
		newRole("staff"),
		newRole("supervisor"),
		newRole("manager"),
		newRole("director"),
		newRole("admin"),
	}

	permissions := []models.Permission{
		newPermission("manageLeaveRequest"),
		newPermission("seeAllLeaveRequest"),

		newPermission("manageTripRequest"),
		newPermission("seeAllTripRequest"),

		newPermission("manageUsers"),
		newPermission("manageEmployee"),

		newPermission("createAttendanceQR"),
		newPermission("seeAllAttendance"),
	}

	for _, role := range roles {
		err = gorm.G[models.Role](db).Create(context.Background(), &role)
		if err != nil {
			log.Fatalf("Failed to seed: %s", err)
		}
	}

	for _, perm := range permissions {
		err = gorm.G[models.Permission](db).Create(context.Background(), &perm)
		if err != nil {
			log.Fatalf("Failed to seed: %s", err)
		}
	}

	permMap := map[string]models.Permission{}
	for _, p := range permissions {
		permMap[p.Codename] = p
	}

	rolePermissions := map[int][]string{
		1: {"manageLeaveRequest", "manageTripRequest"},
		2: {"manageLeaveRequest", "manageTripRequest", "manageEmployee", "createAttendanceQR", "seeAllAttendance"},
		3: {"manageLeaveRequest", "manageTripRequest"},
		4: {"manageLeaveRequest", "manageTripRequest", "manageUsers", "createAttendanceQR", "seeAllAttendance", "seeAllLeaveRequest", "seeAllTripRequest"},
	}

	for roleIdx, codenames := range rolePermissions {
		for _, codename := range codenames {
			rp := models.RolePermission{
				Base:         models.Base{ID: uuid.NewString()},
				RoleID:       roles[roleIdx].ID,
				PermissionID: permMap[codename].ID,
			}
			err = gorm.G[models.RolePermission](db).Create(context.Background(), &rp)
			if err != nil {
				log.Fatalf("Failed to seed role permission: %s", err)
			}
		}
	}

	dummyUsers := []models.User{
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Hafizryandin Haykal",
			LastName:  "Matondang",
			RoleID:    roles[4].ID,
			Email:     "admin@hrconnect.org",
			Password:  hash("Admin@123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Fathir",
			LastName:  "RIH",
			RoleID:    roles[3].ID,
			Email:     "director@hrconnect.org",
			Password:  hash("Director@123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Nopal",
			LastName:  "Pradana",
			RoleID:    roles[2].ID,
			Email:     "manager@hrconnect.org",
			Password:  hash("Manager@123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Haidar",
			LastName:  "Zahran",
			RoleID:    roles[1].ID,
			Email:     "supervisor@hrconnect.org",
			Password:  hash("Supervisor@123"),
		},
		{
			Base: models.Base{
				ID: uuid.NewString(),
			},
			FirstName: "Cecep",
			LastName:  "Wijaya",
			RoleID:    roles[0].ID,
			Email:     "staff@hrconnect.org",
			Password:  hash("Staff@123"),
		},
	}

	for _, user := range dummyUsers {
		err = gorm.G[models.User](db).Create(context.Background(), &user)
		if err != nil {
			log.Fatalf("Failed to seed: %s", err)
		}
	}
}

func newRole(name string) models.Role {
	return models.Role{
		Base: models.Base{
			ID: uuid.NewString(),
		},
		Name: name,
	}
}

func newPermission(codename string) models.Permission {
	return models.Permission{
		Base: models.Base{
			ID: uuid.NewString(),
		},
		Codename: codename,
	}
}

func hash(p string) string {
	h, err := lib.HashPassword(p)
	if err != nil {
		log.Fatalf("Failed to hash password: %s", err)
	}
	return string(h)
}
