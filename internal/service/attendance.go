package service

import (
	"context"

	"github.com/hyoureii/hrbackend/gen/attendance/v1"
	"gorm.io/gorm"
)

type AttendanceServiceServer struct {
	db *gorm.DB
	attendance.UnimplementedAttendanceServiceServer
}

func NewAttendanceServiceServer(db *gorm.DB) *AttendanceServiceServer {
	return &AttendanceServiceServer{db: db}
}

func (s AttendanceServiceServer) CheckIn(c context.Context, r *attendance.CheckInRequest) (*attendance.CheckInResponse, error) {
	return &attendance.CheckInResponse{}, nil
}

func (s AttendanceServiceServer) CheckOut(c context.Context, r *attendance.CheckOutRequest) (*attendance.CheckOutResponse, error) {
	return &attendance.CheckOutResponse{}, nil
}

func (s AttendanceServiceServer) Generate(c context.Context, r *attendance.GenerateRequest) (*attendance.GenerateResponse, error) {

	return &attendance.GenerateResponse{
		Url: "https://hrconnect.hyourei.xyz",
	}, nil
}

func (s AttendanceServiceServer) Today(c context.Context, r *attendance.TodayRequest) (*attendance.TodayResponse, error) {

	return &attendance.TodayResponse{
		Url: "https://hrconnect.hyourei.xyz",
	}, nil
}

func (s AttendanceServiceServer) GetAllAttendance(c context.Context, r *attendance.GetAllAttendanceRequest) (*attendance.GetAllAttendanceResponse, error) {
	return &attendance.GetAllAttendanceResponse{}, nil
}

func (s AttendanceServiceServer) GetAttendanceById(c context.Context, r *attendance.GetAttendanceByIdRequest) (*attendance.GetAttendanceByIdResponse, error) {
	return &attendance.GetAttendanceByIdResponse{}, nil
}

func (s AttendanceServiceServer) GetCurrent(c context.Context, r *attendance.GetCurrentRequest) (*attendance.GetCurrentResponse, error) {
	return &attendance.GetCurrentResponse{}, nil
}
