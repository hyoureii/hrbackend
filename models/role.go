package models

type Role struct {
	Base
	Name string `gorm:"not null;unique"`
}

type Permission struct {
	Base
	Codename string `gorm:"not null;unique"`
}

type RolePermission struct {
	Base
	RoleID       string     `gorm:"type:uuid;not null"`
	Role         Role       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PermissionID string     `gorm:"type:uuid;not null"`
	Permission   Permission `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
