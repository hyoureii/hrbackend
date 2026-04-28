package models

import (
	"time"

	"gorm.io/gorm"
)


type UserRole int

const (
	Admin  UserRole = iota
	Director
	Manager
	Supervisor
	Staff
)

type User struct {
	gorm.Model
	FirstName string `gorm:"not null"`
	LastName string `gorm:"not null"`
	Role UserRole `gorm:"not null"`
	AvatarURL string
	IsActive string `gorm:"default=true"`
	Email string `gorm:"not null;unique"`
	Password string `gorm:"not null"`
}

type RefreshToken struct {
	gorm.Model
	TokenHash string `gorm:"not null;unique"`
	ExpiredAt time.Time
	UserID uint
	User User
}
