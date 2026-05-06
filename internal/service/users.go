package service

import (
	"context"
	"errors"

	usersv1 "github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type UsersServiceServer struct {
	db *gorm.DB
	usersv1.UnimplementedUsersServiceServer
}

func NewUsersServiceServer(db *gorm.DB) *UsersServiceServer {
	return &UsersServiceServer{db: db}
}

func (srv UsersServiceServer) Register(c context.Context, r *usersv1.RegisterRequest) (*usersv1.RegisterResponse, error) {
	hash, err := lib.HashPassword(r.Password)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
		FirstName: r.Data.FirstName,
		LastName: r.Data.LastName,
		Role: r.Data.Role,
		AvatarURL: *r.Data.AvatarUrl,
		Email: r.Data.Email,
		Password: string(hash),
	}
	err = gorm.G[models.User](srv.db).Create(c, newUser)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) { return nil, status.Error(codes.AlreadyExists, "User already exists")}
		return nil, err
	}

	return &usersv1.RegisterResponse{}, nil
}

func (srv UsersServiceServer) ListAll(c context.Context, r *usersv1.ListAllRequest) (*usersv1.ListAllResponse, error) {
	// TODO: implement
	return &usersv1.ListAllResponse{}, nil
}

func (srv UsersServiceServer) GetById(c context.Context, r *usersv1.GetByIdRequest) (*usersv1.GetByIdResponse, error) {
	// TODO: implement
	return &usersv1.GetByIdResponse{}, nil
}

func (srv UsersServiceServer) Me(c context.Context, r *usersv1.MeRequest) (*usersv1.MeResponse, error) {
	// TODO: implement
	return &usersv1.MeResponse{}, nil
}

func (srv UsersServiceServer) Update(c context.Context, r *usersv1.UpdateRequest) (*usersv1.UpdateResponse, error) {
	// TODO: implement
	return &usersv1.UpdateResponse{}, nil
}

func (srv UsersServiceServer) Delete(c context.Context, r *usersv1.DeleteRequest) (*usersv1.DeleteResponse, error) {
	// TODO: implement
	return &usersv1.DeleteResponse{}, nil
}
