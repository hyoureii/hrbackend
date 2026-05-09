package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
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

func (s UsersServiceServer) Register(c context.Context, r *usersv1.RegisterRequest) (*usersv1.RegisterResponse, error) {
	hash, err := lib.HashPassword(r.Password)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	newUser := &models.User{
		Base: models.Base{
			ID: id.String(),
		},
		FirstName: r.Data.FirstName,
		LastName:  r.Data.LastName,
		Role:      r.Data.Role,
		AvatarURL: r.Data.AvatarUrl,
		Email:     r.Data.Email,
		Password:  string(hash),
	}
	err = gorm.G[models.User](s.db).Create(c, newUser)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, "User already exists")
		}
		return nil, err
	}

	return &usersv1.RegisterResponse{}, nil
}

func (s UsersServiceServer) GetAllUsers(c context.Context, r *usersv1.GetAllUsersRequest) (*usersv1.GetAllUsersResponse, error) {
	users, err := gorm.G[models.User](s.db).Find(c)
	if err != nil {
		return nil, err
	}

	userList := make([]*usersv1.User, len(users))
	for i, user := range users {
		userList[i] = &usersv1.User{
			Id: user.ID,
			Data: &usersv1.User_Data{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role,
				AvatarUrl: user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	return &usersv1.GetAllUsersResponse{User: userList}, nil
}

func (s UsersServiceServer) GetUserById(c context.Context, r *usersv1.GetUserByIdRequest) (*usersv1.GetUserByIdResponse, error) {
	user, err := gorm.G[models.User](s.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &usersv1.GetUserByIdResponse{
		User: &usersv1.User{
			Id: user.ID,
			Data: &usersv1.User_Data{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role,
				AvatarUrl: user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

func (s UsersServiceServer) Me(c context.Context, r *usersv1.MeRequest) (*usersv1.MeResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.Claims)
	user, err := gorm.G[models.User](s.db).Where("id = ?", claims.Subject).First(c)
	if err != nil {
		return nil, err
	}

	return &usersv1.MeResponse{
		User: &usersv1.User{
			Id: user.ID,
			Data: &usersv1.User_Data{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role,
				AvatarUrl: user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

func (s UsersServiceServer) Update(c context.Context, r *usersv1.UpdateRequest) (*usersv1.UpdateResponse, error) {
	_, err := gorm.G[models.User](s.db).Where("id = ?", r.Id).Select("email", "first_name", "last_name", "avatar_url", "role").Updates(c, models.User{
		Email:     r.Data.Email,
		FirstName: r.Data.FirstName,
		LastName:  r.Data.LastName,
		AvatarURL: r.Data.AvatarUrl,
		Role:      r.Data.Role,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &usersv1.UpdateResponse{}, nil
}

func (s UsersServiceServer) Delete(c context.Context, r *usersv1.DeleteRequest) (*usersv1.DeleteResponse, error) {
	_, err := gorm.G[models.User](s.db).Where("id = ?", r.Id).Delete(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &usersv1.DeleteResponse{}, nil
}
