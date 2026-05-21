package models

type Attendance struct {
	Base
	UserID    string `gorm:"type:uuid;not null;index:idx_attendance,unique"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ScannedAt int64  `gorm:"not null;index:idx_attendance,unique"`
	Payload   string `gorm:"not null"`
}

type QrCode struct {
	Base
	ExpiredAt int64  `gorm:"not null"`
	Payload   string `gorm:"not null"`
}
