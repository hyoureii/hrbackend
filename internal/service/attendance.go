package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hyoureii/hrbackend/gen/attendance/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AttendanceServiceServer struct {
	db  *gorm.DB
	rdb *redis.Client
	attendance.UnimplementedAttendanceServiceServer
}

func NewAttendanceServiceServer(db *gorm.DB, rdb *redis.Client) *AttendanceServiceServer {
	return &AttendanceServiceServer{db: db, rdb: rdb}
}

func (s AttendanceServiceServer) CheckIn(c context.Context, r *attendance.CheckInRequest) (*attendance.CheckInResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	attClaims, err := lib.ValidateAttendanceJwt(r.Payload)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid or expired QR code")
	}

	if attClaims.Type != lib.CheckIn {
		return nil, status.Error(codes.InvalidArgument, "this QR is not for check-in")
	}

	qrCode, err := gorm.G[models.QrCode](s.db).Where("id = ?", attClaims.ID).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "QR code not found or has been replaced")
		}
		return nil, err
	}

	now := time.Now()
	hour := now.Hour()
	if hour < 8 || hour >= 9 {
		return nil, status.Error(codes.FailedPrecondition, "outside check-in time")
	}

	nonceKey := "nonce:" + attClaims.ID + `@` + attClaims.Subject 
	exists, err := s.rdb.Exists(c, nonceKey).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check nonce")
	}
	if exists == 1 {
		return nil, status.Error(codes.AlreadyExists, "user already scanned this qr code")
	}

	ttl := time.Until(time.Unix(qrCode.ExpiredAt, 0))
	err = s.rdb.Set(c, nonceKey, "1", ttl).Err()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to store nonce")
	}

	att := &models.Attendance{
		Base:        models.Base{ID: uuid.New().String()},
		EmployeeID:  claims.Subject,
		WorkDay:     time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix(),
		CheckInAt:   now.Unix(),
		CheckInQrID: qrCode.ID,
	}
	err = gorm.G[models.Attendance](s.db).Omit("check_out_qr_id").Create(c, att)
	if err != nil {
		if errors.Is(err, gorm.ErrCheckConstraintViolated) {
			return nil, status.Error(codes.AlreadyExists, "already checked in today")
		}
		return nil, err
	}

	return &attendance.CheckInResponse{}, nil
}

func (s AttendanceServiceServer) CheckOut(c context.Context, r *attendance.CheckOutRequest) (*attendance.CheckOutResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	attClaims, err := lib.ValidateAttendanceJwt(r.Payload)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid or expired QR code")
	}

	if attClaims.Type != lib.CheckOut {
		return nil, status.Error(codes.InvalidArgument, "this QR is not for check-out")
	}

	qrCode, err := gorm.G[models.QrCode](s.db).Where("id = ?", attClaims.ID).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "QR code not found or has been replaced")
		}
		return nil, err
	}

	now := time.Now()
	hour := now.Hour()
	if hour < 16 || hour >= 17 {
		return nil, status.Error(codes.FailedPrecondition, "outside check-out time")
	}

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	att, err := gorm.G[models.Attendance](s.db).
		Where("employee_id = ?", claims.Subject).
		Where("work_day >= ? AND work_day < ?", startOfDay.Unix(), endOfDay.Unix()).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "must check in first")
		}
		return nil, err
	}

	if att.CheckOutAt != 0 {
		return nil, status.Error(codes.AlreadyExists, "already checked out today")
	}

	nonceKey := "nonce:" + attClaims.ID + `@` + attClaims.Subject
	exists, err := s.rdb.Exists(c, nonceKey).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check nonce")
	}
	if exists == 1 {
		return nil, status.Error(codes.AlreadyExists, "user already scanned this qr code")
	}

	ttl := time.Until(time.Unix(qrCode.ExpiredAt, 0))
	err = s.rdb.Set(c, nonceKey, "1", ttl).Err()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to store nonce")
	}

	_, err = gorm.G[models.Attendance](s.db).Where("id = ?", att.ID).Updates(c, models.Attendance{
		CheckOutAt:   now.Unix(),
		CheckOutQrID: qrCode.ID,
	})
	if err != nil {
		return nil, err
	}

	return &attendance.CheckOutResponse{}, nil
}

func (s AttendanceServiceServer) Generate(c context.Context, r *attendance.GenerateRequest) (*attendance.GenerateResponse, error) {
	now := time.Now()
	hour := now.Hour()
	expDur := time.Minute * 5

	var purpose models.QrType
	switch {
	case hour >= 8 && hour < 9:
		purpose = models.CheckIn
	case hour >= 16 && hour < 17:
		purpose = models.CheckOut
	default:
		return nil, status.Error(codes.FailedPrecondition, "Outside check in/out time")
	}

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	_, err := gorm.G[models.QrCode](s.db).
		Where("purpose = ?", purpose).
		Where("issued_at >= ? AND issued_at < ?", startOfDay.Unix(), endOfDay.Unix()).
		Delete(c)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate QR code ID")
	}

	exp := now.Add(expDur)
	err = gorm.G[models.QrCode](s.db).Create(c, &models.QrCode{
		Base:      models.Base{ID: id.String()},
		IssuedAt:  now.Unix(),
		ExpiredAt: exp.Unix(),
		Purpose:   purpose,
	})
	if err != nil {
		return nil, err
	}

	png, err := lib.GenerateAttendanceQr(lib.AttendanceClaimType(purpose), exp, id.String())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate QR code")
	}

	url := "https://hr.hyourei.xyz/qr/" + id.String()
	err = s.rdb.Set(c, "qr:"+id.String(), png, expDur).Err()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to set cache for qr")
	}

	return &attendance.GenerateResponse{Url: url}, nil
}

func (s AttendanceServiceServer) GetAllAttendance(c context.Context, r *attendance.GetAllAttendanceRequest) (*attendance.GetAllAttendanceResponse, error) {
	records, err := gorm.G[models.Attendance](s.db).Find(c)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]*attendance.Attendance)
	for _, rec := range records {
		grouped[rec.EmployeeID] = append(grouped[rec.EmployeeID], &attendance.Attendance{
			Workday:    rec.WorkDay,
			CheckInAt:  rec.CheckInAt,
			CheckOutAt: rec.CheckOutAt,
		})
	}

	var userAtt []*attendance.GetAllAttendanceResponse_UserAttendance
	for uid, atts := range grouped {
		userAtt = append(userAtt, &attendance.GetAllAttendanceResponse_UserAttendance{
			UserId:     uid,
			Attendance: atts,
		})
	}

	return &attendance.GetAllAttendanceResponse{UserAttendance: userAtt}, nil
}

func (s AttendanceServiceServer) GetAttendanceById(c context.Context, r *attendance.GetAttendanceByIdRequest) (*attendance.GetAttendanceByIdResponse, error) {
	records, err := gorm.G[models.Attendance](s.db).Where("employee_id = ?", r.Id).Find(c)
	if err != nil {
		return nil, err
	}

	var atts []*attendance.Attendance
	for _, rec := range records {
		atts = append(atts, &attendance.Attendance{
			Workday:    rec.WorkDay,
			CheckInAt:  rec.CheckInAt,
			CheckOutAt: rec.CheckOutAt,
		})
	}

	return &attendance.GetAttendanceByIdResponse{Attendance: atts}, nil
}

func (s AttendanceServiceServer) GetCurrent(c context.Context, r *attendance.GetCurrentRequest) (*attendance.GetCurrentResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	records, err := gorm.G[models.Attendance](s.db).Where("employee_id = ?", claims.Subject).Find(c)
	if err != nil {
		return nil, err
	}

	var atts []*attendance.Attendance
	for _, rec := range records {
		atts = append(atts, &attendance.Attendance{
			Workday:    rec.WorkDay,
			CheckInAt:  rec.CheckInAt,
			CheckOutAt: rec.CheckOutAt,
		})
	}

	return &attendance.GetCurrentResponse{Attendance: atts}, nil
}
