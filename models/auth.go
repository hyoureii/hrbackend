package models

import (
	"time"

	authv1 "github.com/hyoureii/hrbackend/gen"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	FirstName string  `gorm:"not null"`
	LastName  string  `gorm:"not null"`
	Role      authv1.Role `gorm:"not null;default:0"`
	AvatarURL string
	IsActive  bool   `gorm:"not null;default:true"`
	Email     string `gorm:"not null;unique"`
	Password  string `gorm:"not null"`
}

type RefreshToken struct {
	gorm.Model
	TokenHash string    `gorm:"not null;unique"`
	ExpiredAt time.Time `gorm:"not null"`
	UserID    uint      `gorm:"not null"`
	User      User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
