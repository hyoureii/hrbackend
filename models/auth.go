package models

import (
	"github.com/hyoureii/hrbackend/gen/users/v1"
)

type User struct {
	Base
	FirstName string       `gorm:"not null"`
	LastName  string       `gorm:"not null"`
	Role      users.Role `gorm:"not null;default:0"`
	AvatarURL *string
	IsActive  bool   `gorm:"not null;default:true"`
	Email     string `gorm:"not null;unique"`
	Password  string `gorm:"not null"`
}

type RefreshToken struct {
	Base
	TokenHash string `gorm:"not null;unique"`
	ExpiredAt int64  `gorm:"not null"`
	UserID    string `gorm:"type:uuid;not null"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
