package service

import (
	"context"
	"slices"
	"time"

	dashboard "github.com/hyoureii/hrbackend/gen/dashboard/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"gorm.io/gorm"
)

type DashboardServiceServer struct {
	db *gorm.DB
	dashboard.UnimplementedDashboardServiceServer
}

func NewDashboardServiceServer(db *gorm.DB) *DashboardServiceServer {
	return &DashboardServiceServer{db: db}
}

func (s DashboardServiceServer) Dashboard(ctx context.Context, req *dashboard.DashboardRequest) (*dashboard.DashboardResponse, error) {
	claims := ctx.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := startOfDay.Add(-30 * 24 * time.Hour)

	totalWeekdays := 0
	for d := cutoff; !d.After(now); d = d.Add(24 * time.Hour) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			totalWeekdays++
		}
	}

	records, err := gorm.G[models.Attendance](s.db).
		Where("employee_id = ? AND work_day >= ?", claims.Subject, cutoff.Unix()).
		Find(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, r := range records {
		t := time.Unix(r.WorkDay, 0)
		if wd := t.Weekday(); wd != time.Saturday && wd != time.Sunday {
			key := t.Format("2006-01-02")
			seen[key] = struct{}{}
		}
	}

	attended := len(seen)
	attendanceRate := int32(0)
	if totalWeekdays > 0 {
		attendanceRate = int32(attended * 100 / totalWeekdays)
	}

	var pendingLeave int64
	s.db.Model(&models.Leave{}).
		Where("requester_id = ? AND status = ?", claims.Subject, models.Pending).
		Count(&pendingLeave)

	var pendingTrip int64
	s.db.Model(&models.Trip{}).
		Where("requester_id = ? AND status = ?", claims.Subject, models.Pending).
		Count(&pendingTrip)

	res := &dashboard.DashboardResponse{
		AttendanceRate: attendanceRate,
		PendingLeave:   int32(pendingLeave),
		PendingTrip:    int32(pendingTrip),
	}

	if slices.Contains(claims.Perms, "manageUsers") {
		var totalUsers int64
		s.db.Model(&models.User{}).Where("is_active = ?", true).Count(&totalUsers)
		n := int32(totalUsers)
		res.TotalUser = &n
	}

	return res, nil
}
