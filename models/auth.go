package models

type User struct {
	Base
	FirstName string `gorm:"not null"`
	LastName  string `gorm:"not null"`
	RoleID    string `gorm:"type:uuid;not null"`
	Role      Role   `gorm:"not null"`
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
	Jti       string `gorm:"type:uuid;not null;unique"`
}
