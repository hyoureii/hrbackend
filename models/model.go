package models

type Base struct {
	ID        string `gorm:"type:uuid;primarykey"`
	CreatedAt int64
	UpdatedAt int64
}
