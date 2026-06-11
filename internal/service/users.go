package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UsersServiceServer struct {
	db *gorm.DB
	users.UnimplementedUsersServiceServer
}

func NewUsersServiceServer(db *gorm.DB) *UsersServiceServer {
	return &UsersServiceServer{db: db}
}

func (s UsersServiceServer) Register(c context.Context, r *users.RegisterRequest) (*users.RegisterResponse, error) {
	valid, err := lib.ValidatePassword(r.Password)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "Password format invalid")
	}

	hash, err := lib.HashPassword(r.Password)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	roleId, err := getRoleID(c, r.Data.Role, s.db)
	if err != nil {
		return nil, err
	}
	newUser := &models.User{
		Base: models.Base{
			ID: id.String(),
		},
		FirstName: r.Data.FirstName,
		LastName:  r.Data.LastName,
		RoleID:    roleId,
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

	return &users.RegisterResponse{}, nil
}

func (s UsersServiceServer) GetAllUsers(c context.Context, r *users.GetAllUsersRequest) (*users.GetAllUsersResponse, error) {
	usersRes, err := gorm.G[models.User](s.db).Joins(clause.LeftJoin.Association("Role"), nil).Find(c)
	if err != nil {
		return nil, err
	}

	userList := make([]*users.User, len(usersRes))
	for i, user := range usersRes {
		userList[i] = &users.User{
			Id: user.ID,
			Data: &users.User_Data{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role.Name,
				AvatarUrl: user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	return &users.GetAllUsersResponse{User: userList}, nil
}

func (s UsersServiceServer) GetUserById(c context.Context, r *users.GetUserByIdRequest) (*users.GetUserByIdResponse, error) {
	user, err := gorm.G[models.User](s.db).Joins(clause.LeftJoin.Association("Role"), nil).Where("users.id = ?", r.Id).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &users.GetUserByIdResponse{
		User: &users.User{
			Id: user.ID,
			Data: &users.User_Data{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role.Name,
				AvatarUrl: user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

func (s UsersServiceServer) Me(c context.Context, r *users.MeRequest) (*users.MeResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)
	user, err := gorm.G[models.User](s.db).Joins(clause.LeftJoin.Association("Role"), nil).Where("users.id = ?", claims.Subject).First(c)
	if err != nil {
		return nil, err
	}

	return &users.MeResponse{
		User: &users.User{
			Id: user.ID,
			Data: &users.User_Data{
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Role:      user.Role.Name,
				AvatarUrl: user.AvatarURL,
			},
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

func (s UsersServiceServer) Deactivate(c context.Context, r *users.DeactivateRequest) (*users.DeactivateResponse, error) {
	user, err := gorm.G[models.User](s.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, status.Error(codes.InvalidArgument, "User is non-active")
	}

	_, err = gorm.G[models.User](s.db).Where("id = ?", user.ID).Update(c, "is_active", false)
	if err != nil {
		return nil, err
	}

	return &users.DeactivateResponse{}, nil
}

func (s UsersServiceServer) Activate(c context.Context, r *users.ActivateRequest) (*users.ActivateResponse, error) {
	user, err := gorm.G[models.User](s.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	if user.IsActive {
		return nil, status.Error(codes.InvalidArgument, "User is active")
	}

	_, err = gorm.G[models.User](s.db).Where("id = ?", user.ID).Update(c, "is_active", true)
	if err != nil {
		return nil, err
	}

	return &users.ActivateResponse{}, nil
}

func (s UsersServiceServer) Update(c context.Context, r *users.UpdateRequest) (*users.UpdateResponse, error) {
	roleId, err := getRoleID(c, r.Data.Role, s.db)
	if err != nil {
		return nil, err
	}
	_, err = gorm.G[models.User](s.db).Where("id = ?", r.Id).Select("email", "first_name", "last_name", "avatar_url", "role_id").Updates(c, models.User{
		Email:     r.Data.Email,
		FirstName: r.Data.FirstName,
		LastName:  r.Data.LastName,
		AvatarURL: r.Data.AvatarUrl,
		RoleID:    roleId,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &users.UpdateResponse{}, nil
}

func (s UsersServiceServer) Delete(c context.Context, r *users.DeleteRequest) (*users.DeleteResponse, error) {
	_, err := gorm.G[models.User](s.db).Where("id = ?", r.Id).Delete(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &users.DeleteResponse{}, nil
}

func getRoleID(c context.Context, roleName string, db *gorm.DB) (string, error) {
	role, err := gorm.G[models.Role](db).Where("name = ?", roleName).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			def, err := gorm.G[models.Role](db).Where("name = ?", "Staff").First(c)
			if err != nil {
				return "", err
			}
			return def.ID, nil
		}
		return "", err
	}

	return role.ID, nil
}
