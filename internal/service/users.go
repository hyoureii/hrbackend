package service

import (
	"context"
	"errors"

	usersv1 "github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
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
	users, err := gorm.G[models.User](srv.db).Find(c)
	if err != nil {
		return nil, err
	}

	userList := make([]*usersv1.UserFull, len(users))
	for i, user := range users {
		userList[i] = &usersv1.UserFull{
			Id: int64(user.ID),
			Data: &usersv1.User{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role,
				AvatarUrl: &user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		}
	}

	return &usersv1.ListAllResponse{ User: userList }, nil
}

func (srv UsersServiceServer) GetById(c context.Context, r *usersv1.GetByIdRequest) (*usersv1.GetByIdResponse, error) {
	user, err := gorm.G[models.User](srv.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		return nil, err
	}

	return &usersv1.GetByIdResponse{
		User: &usersv1.UserFull{
			Id: int64(user.ID),
			Data: &usersv1.User{
				Email: user.Email,
				FirstName: user.FirstName,
				LastName: user.LastName,
				Role: user.Role,
				AvatarUrl: &user.AvatarURL,
			},
			IsActive: user.IsActive,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (srv UsersServiceServer) Me(c context.Context, r *usersv1.MeRequest) (*usersv1.MeResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.Claims)
	user, err := gorm.G[models.User](srv.db).Where("id = ?", claims.Subject).First(c)
	if err != nil {
		return nil, err
	}

	return &usersv1.MeResponse{
		User: &usersv1.UserFull{
			Id: int64(user.ID),
			Data: &usersv1.User{
				Email: user.Email,
				FirstName: user.FirstName,
				LastName: user.LastName,
				Role: user.Role,
				AvatarUrl: &user.AvatarURL,
			},
			IsActive: user.IsActive,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (srv UsersServiceServer) Update(c context.Context, r *usersv1.UpdateRequest) (*usersv1.UpdateResponse, error) {
	_, err := gorm.G[models.User](srv.db).Where("id = ?", r.Id).Select("email", "first_name", "last_name", "avatar_url", "role").Updates(c, models.User{
		Email: r.Data.Email,
		FirstName: r.Data.FirstName,
		LastName: r.Data.LastName,
		AvatarURL: *r.Data.AvatarUrl,
		Role: r.Data.Role,
	})
	if err != nil {
		return nil, err
	}

	return &usersv1.UpdateResponse{}, nil
}

func (srv UsersServiceServer) Delete(c context.Context, r *usersv1.DeleteRequest) (*usersv1.DeleteResponse, error) {
	_, err := gorm.G[models.User](srv.db).Where("id = ?", r.Id).Delete(c)
	if err != nil {
		return nil, err
	}

	return &usersv1.DeleteResponse{}, nil
}
