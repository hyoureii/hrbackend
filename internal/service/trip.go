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

type TripServiceServer struct {
	db *gorm.DB
	request.UnimplementedTripServiceServer
}

func NewTripServiceServer(db *gorm.DB) *TripServiceServer {
	return &TripServiceServer{db: db}
}

func (s TripServiceServer) NewTrip(c context.Context, r *request.NewTripRequest) (*request.NewTripResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	tripType := models.TripType(r.Request.Type)
	if !slices.Contains(models.TripTypes, tripType) {
		return nil, status.Error(codes.InvalidArgument, "Invalid trip type")
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

	newTrip := &models.Trip{
		Request: models.Request{
			Base:        models.Base{ID: id.String()},
			Description: r.Request.Description,
			StartDate:   r.Request.StartDate,
			EndDate:     r.Request.EndDate,
			Status:      models.Pending,
			RequesterID: claims.Subject,
		},
		Type: tripType,
	}

	err = gorm.G[models.Trip](s.db).Create(c, newTrip)
	if err != nil {
		return nil, err
	}

	return &request.NewTripResponse{}, nil
}

func (s TripServiceServer) GetAllTrips(c context.Context, r *request.GetAllTripsRequest) (*request.GetAllTripsResponse, error) {
	q := gorm.G[models.Trip](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil)

	if r.ByUserId != nil && *r.ByUserId != "" {
		q = q.Where("trips.requester_id = ?", *r.ByUserId)
	}
	if r.Status != nil && *r.Status != "" {
		q = q.Where("trips.status = ?", *r.Status)
	}

	trips, err := q.Find(c)
	if err != nil {
		return nil, err
	}

	res := make([]*request.Request, len(trips))
	for i, l := range trips {
		res[i] = tripToProto(l)
	}

	return &request.GetAllTripsResponse{Requests: res}, nil
}

func (s TripServiceServer) GetCurrentTrips(c context.Context, r *request.GetCurrentTripsRequest) (*request.GetCurrentTripsResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	trips, err := gorm.G[models.Trip](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil).
		Where("trips.requester_id = ?", claims.Subject).
		Find(c)
	if err != nil {
		return nil, err
	}

	res := make([]*request.Request, len(trips))
	for i, l := range trips {
		res[i] = tripToProto(l)
	}

	return &request.GetCurrentTripsResponse{Requests: res}, nil
}

func (s TripServiceServer) GetAllPendingTrips(c context.Context, r *request.GetAllPendingTripsRequest) (*request.GetAllPendingTripsResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	roleBelow := lib.RoleBelow(claims.Role)
	if roleBelow == "" {
		return &request.GetAllPendingTripsResponse{}, nil
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
		return &request.GetAllPendingTripsResponse{}, nil
	}

	trips, err := gorm.G[models.Trip](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil).
		Where("trips.status = ? AND trips.requester_id IN ?", models.Pending, userIDs).
		Find(c)
	if err != nil {
		return nil, err
	}

	res := make([]*request.Request, len(trips))
	for i, l := range trips {
		res[i] = tripToProto(l)
	}

	return &request.GetAllPendingTripsResponse{Requests: res}, nil
}

func (s TripServiceServer) GetTripById(c context.Context, r *request.GetTripByIdRequest) (*request.GetTripByIdResponse, error) {
	trip, err := gorm.G[models.Trip](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Joins(clause.LeftJoin.Association("Requester.Role"), nil).
		Joins(clause.LeftJoin.Association("Approver"), nil).
		Joins(clause.LeftJoin.Association("Approver.Role"), nil).
		Where("trips.id = ?", r.Id).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Trip request not found")
		}
		return nil, err
	}

	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)
	perms := c.Value(middleware.PermsKey).([]string)

	if slices.Contains(perms, "manageTripRequest") && !slices.Contains(perms, "seeAllTripRequest") {
		requester, err := gorm.G[models.User](s.db).
			Joins(clause.LeftJoin.Association("Role"), nil).
			Where("users.id = ?", trip.RequesterID).
			First(c)
		if err != nil {
			return nil, err
		}
		if !lib.CanManage(claims.Role, requester.Role.Name) {
			return nil, status.Error(codes.PermissionDenied, "You cannot view requests from this role")
		}
	}

	return &request.GetTripByIdResponse{Request: tripToProto(trip)}, nil
}

func (s TripServiceServer) UpdateTrip(c context.Context, r *request.UpdateTripRequest) (*request.UpdateTripResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	trip, err := gorm.G[models.Trip](s.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Trip request not found")
		}
		return nil, err
	}

	if trip.RequesterID != claims.Subject {
		return nil, status.Error(codes.PermissionDenied, "You can only update your own requests")
	}

	tripType := models.TripType(r.Request.Type)
	if !slices.Contains(models.TripTypes, tripType) {
		return nil, status.Error(codes.InvalidArgument, "Invalid trip type")
	}

	_, err = gorm.G[models.Trip](s.db).Where("id = ?", r.Id).Select("description", "start_date", "end_date", "type").Updates(c, models.Trip{
		Request: models.Request{
			Description: r.Request.Description,
			StartDate:   r.Request.StartDate,
			EndDate:     r.Request.EndDate,
		},
		Type: tripType,
	})
	if err != nil {
		return nil, err
	}

	return &request.UpdateTripResponse{}, nil
}

func (s TripServiceServer) ApproveTrip(c context.Context, r *request.ApproveTripRequest) (*request.ApproveTripResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	trip, err := gorm.G[models.Trip](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Where("trips.id = ?", r.Id).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Trip request not found")
		}
		return nil, err
	}

	if trip.RequesterID == claims.Subject {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage your own request")
	}

	if trip.Status != models.Pending {
		return nil, status.Error(codes.InvalidArgument, "Request is not pending")
	}

	requester, err := gorm.G[models.User](s.db).
		Joins(clause.LeftJoin.Association("Role"), nil).
		Where("users.id = ?", trip.RequesterID).
		First(c)
	if err != nil {
		return nil, err
	}

	if !lib.CanManage(claims.Role, requester.Role.Name) {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage requests from this role")
	}

	_, err = gorm.G[models.Trip](s.db).Where("id = ?", r.Id).Update(c, "status", models.Approved)
	if err != nil {
		return nil, err
	}
	_, err = gorm.G[models.Trip](s.db).Where("id = ?", r.Id).Update(c, "approver_id", claims.Subject)
	if err != nil {
		return nil, err
	}

	return &request.ApproveTripResponse{}, nil
}

func (s TripServiceServer) RejectTrip(c context.Context, r *request.RejectTripRequest) (*request.RejectTripResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)

	trip, err := gorm.G[models.Trip](s.db).
		Joins(clause.LeftJoin.Association("Requester"), nil).
		Where("trips.id = ?", r.Id).
		First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "Trip request not found")
		}
		return nil, err
	}

	if trip.RequesterID == claims.Subject {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage your own request")
	}

	if trip.Status != models.Pending {
		return nil, status.Error(codes.InvalidArgument, "Request is not pending")
	}

	requester, err := gorm.G[models.User](s.db).
		Joins(clause.LeftJoin.Association("Role"), nil).
		Where("users.id = ?", trip.RequesterID).
		First(c)
	if err != nil {
		return nil, err
	}

	if !lib.CanManage(claims.Role, requester.Role.Name) {
		return nil, status.Error(codes.PermissionDenied, "You cannot manage requests from this role")
	}

	_, err = gorm.G[models.Trip](s.db).Where("id = ?", r.Id).Updates(c, models.Trip{
		Request: models.Request{
			Status:       models.Rejected,
			ApproverID:   claims.Subject,
			RejectReason: r.GetReason(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &request.RejectTripResponse{}, nil
}

func tripToProto(l models.Trip) *request.Request {
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
