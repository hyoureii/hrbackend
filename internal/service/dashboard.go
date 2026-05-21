package service

import (
	"github.com/hyoureii/hrbackend/gen/dashboard/v1"
	"gorm.io/gorm"
)

type DashboardServiceServer struct {
	db *gorm.DB
	dashboard.UnimplementedDashboardServiceServer
}
