package models

type Attendance struct {
	Base
	EmployeeID   string `gorm:"type:uuid;not null;index:idx_attendance,unique"`
	Employee     User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	WorkDay      int64  `gorm:"not null;index:idx_attendance,unique"`
	CheckInAt    int64  `gorm:"not null"`
	CheckInQrID  string `gorm:"type:uuid;not null"`
	CheckInQr    QrCode `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CheckOutAt   int64
	CheckOutQrID string `gorm:"type:uuid"`
	CheckOutQr   QrCode `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

type QrType string

const (
	CheckIn  QrType = "checkin"
	CheckOut QrType = "checkout"
)

var QrTypes = []QrType{CheckIn, CheckOut}

type QrCode struct {
	Base
	IssuedAt  int64  `gorm:"not null"`
	ExpiredAt int64  `gorm:"not null"`
	Purpose   QrType `gorm:"not null;type:hr_qr_type"`
}
