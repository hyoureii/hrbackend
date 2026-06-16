package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/hyoureii/hrbackend/gen/request/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LeaveServiceServer struct {
	db *gorm.DB
	request.UnimplementedLeaveServiceServer
}

func NewLeaveServiceServer(db *gorm.DB) *LeaveServiceServer {
	return &LeaveServiceServer{db: db}
}

func (s LeaveServiceServer) NewLeave(c context.Context, r *request.NewLeaveRequest) (*request.NewLeaveResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	leaveType := models.LeaveType(r.Request.Type)
	if !slices.Contains(models.LeaveTypes, leaveType) {
		return nil, status.Error(codes.InvalidArgument, "Invalid leave type")
	}

	start := time.Unix(r.Request.StartDate, 0)
	end := time.Unix(r.Request.EndDate, 0)
	if !start.After(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "Start date must be in the future")
	}
	if !end.After(start) {
		return nil, status.Error(codes.InvalidArgument, "End date must be after start date")
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	newLeave := &models.Leave{
		Request: models.Request{
			Base:        models.Base{ID: id.String()},
			Description: r.Request.Description,
			StartDate:   r.Request.StartDate,
			EndDate:     r.Request.EndDate,
			Status:      models.Pending,
			RequesterID: claims.Subject,
		},
		Type: leaveType,
	}

	err = gorm.G[models.Leave](s.db).Create(c, newLeave)
	if err != nil {
		return nil, err
	}

	return &request.NewLeaveResponse{}, nil
}

func (s LeaveServiceServer) GetAllLeaves(c context.Context, r *request.GetAllLeavesRequest) (*request.GetAllLeavesResponse, error) {
	q := gorm.G[models.Leave](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil)

	if r.ByUserId != nil && *r.ByUserId != "" {
		q = q.Where("leaves.requester_id = ?", *r.ByUserId)
	}
	if r.Status != nil && *r.Status != "" {
		q = q.Where("leaves.status = ?", *r.Status)
	}

	leaves, err := q.Find(c)
	if err != nil {
		return nil, err
	}

	res := make([]*request.Request, len(leaves))
	for i, l := range leaves {
		res[i] = leaveToProto(l)
	}

	return &request.GetAllLeavesResponse{Requests: res}, nil
}

func (s LeaveServiceServer) GetCurrentLeaves(c context.Context, r *request.GetCurrentLeavesRequest) (*request.GetCurrentLeavesResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	leaves, err := gorm.G[models.Leave](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil).
		Where("leaves.requester_id = ?", claims.Subject).
		Find(c)
	if err != nil {
		return nil, err
	}

	res := make([]*request.Request, len(leaves))
	for i, l := range leaves {
		res[i] = leaveToProto(l)
	}

	return &request.GetCurrentLeavesResponse{Requests: res}, nil
}

func (s LeaveServiceServer) GetAllPendingLeaves(c context.Context, r *request.GetAllPendingLeavesRequest) (*request.GetAllPendingLeavesResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	roleBelow := lib.RoleBelow(claims.Role)
	if roleBelow == "" {
		return &request.GetAllPendingLeavesResponse{}, nil
	}

	role, err := gorm.G[models.Role](s.db).Where("name = ?", roleBelow).First(c)
	if err != nil {
		return nil, err
	}

	users, err := gorm.G[models.User](s.db).Where("role_id = ?", role.ID).Find(c)
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	if len(userIDs) == 0 {
		return &request.GetAllPendingLeavesResponse{}, nil
	}

	leaves, err := gorm.G[models.Leave](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil).
		Where("leaves.status = ? AND leaves.requester_id IN ?", models.Pending, userIDs).
		Find(c)
	if err != nil {
		return nil, err
	}

	res := make([]*request.Request, len(leaves))
	for i, l := range leaves {
		res[i] = leaveToProto(l)
	}

	return &request.GetAllPendingLeavesResponse{Requests: res}, nil
}

func (s LeaveServiceServer) GetLeaveById(c context.Context, r *request.GetLeaveByIdRequest) (*request.GetLeaveByIdResponse, error) {
	leave, err := gorm.G[models.Leave](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil).
		Where("leaves.id = ?", r.Id).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Leave request not found")
		}
		return nil, err
	}

	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)
	perms := c.Value(middleware.PermsKey).([]string)

	if slices.Contains(perms, "manageLeaveRequest") && !slices.Contains(perms, "seeAllLeaveRequest") {
		requester, err := gorm.G[models.User](s.db).
			Joins(clause.LeftJoin.Association("Role"), nil).
			Where("users.id = ?", leave.RequesterID).
			First(c)
		if err != nil {
			return nil, err
		}
		if !lib.CanManage(claims.Role, requester.Role.Name) {
			return nil, status.Error(codes.PermissionDenied, "You cannot view requests from this role")
		}
	}

	return &request.GetLeaveByIdResponse{Request: leaveToProto(leave)}, nil
}

func (s LeaveServiceServer) UpdateLeave(c context.Context, r *request.UpdateLeaveRequest) (*request.UpdateLeaveResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	leave, err := gorm.G[models.Leave](s.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Leave request not found")
		}
		return nil, err
	}

	if leave.RequesterID != claims.Subject {
		return nil, status.Error(codes.PermissionDenied, "You can only update your own requests")
	}

	leaveType := models.LeaveType(r.Request.Type)
	if !slices.Contains(models.LeaveTypes, leaveType) {
		return nil, status.Error(codes.InvalidArgument, "Invalid leave type")
	}

	_, err = gorm.G[models.Leave](s.db).Where("id = ?", r.Id).Select("description", "start_date", "end_date", "type").Updates(c, models.Leave{
		Request: models.Request{
			Description: r.Request.Description,
			StartDate:   r.Request.StartDate,
			EndDate:     r.Request.EndDate,
		},
		Type: leaveType,
	})
	if err != nil {
		return nil, err
	}

	return &request.UpdateLeaveResponse{}, nil
}

func (s LeaveServiceServer) ApproveLeave(c context.Context, r *request.ApproveLeaveRequest) (*request.ApproveLeaveResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	leave, err := gorm.G[models.Leave](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Where("leaves.id = ?", r.Id).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Leave request not found")
		}
		return nil, err
	}

	if leave.RequesterID == claims.Subject {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage your own request")
	}

	if leave.Status != models.Pending {
		return nil, status.Error(codes.InvalidArgument, "Request is not pending")
	}

	requester, err := gorm.G[models.User](s.db).
		Joins(clause.LeftJoin.Association("Role"), nil).
		Where("users.id = ?", leave.RequesterID).
		First(c)
	if err != nil {
		return nil, err
	}

	if !lib.CanManage(claims.Role, requester.Role.Name) {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage requests from this role")
	}

	_, err = gorm.G[models.Leave](s.db).Where("id = ?", r.Id).Update(c, "status", models.Approved)
	if err != nil {
		return nil, err
	}
	_, err = gorm.G[models.Leave](s.db).Where("id = ?", r.Id).Update(c, "approver_id", claims.Subject)
	if err != nil {
		return nil, err
	}

	return &request.ApproveLeaveResponse{}, nil
}

func (s LeaveServiceServer) RejectLeave(c context.Context, r *request.RejectLeaveRequest) (*request.RejectLeaveResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	leave, err := gorm.G[models.Leave](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Where("leaves.id = ?", r.Id).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Leave request not found")
		}
		return nil, err
	}

	if leave.RequesterID == claims.Subject {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage your own request")
	}

	if leave.Status != models.Pending {
		return nil, status.Error(codes.InvalidArgument, "Request is not pending")
	}

	requester, err := gorm.G[models.User](s.db).
		Joins(clause.LeftJoin.Association("Role"), nil).
		Where("users.id = ?", leave.RequesterID).
		First(c)
	if err != nil {
		return nil, err
	}

	if !lib.CanManage(claims.Role, requester.Role.Name) {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage requests from this role")
	}

	_, err = gorm.G[models.Leave](s.db).Where("id = ?", r.Id).Updates(c, models.Leave{
		Request: models.Request{
			Status:       models.Rejected,
			ApproverID:   claims.Subject,
			RejectReason: r.GetReason(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &request.RejectLeaveResponse{}, nil
}

func leaveToProto(l models.Leave) *request.Request {
	return &request.Request{
		Data: &request.Request_Data{
			Description: l.Description,
			StartDate:   l.StartDate,
			EndDate:     l.EndDate,
			Type:        string(l.Type),
		},
		Id:           l.ID,
		Status:       string(l.Status),
		Approver:     lib.UserDataToProto(l.Approver),
		RejectReason: l.RejectReason,
		Requester:    lib.UserDataToProto(l.Requester),
	}
}
