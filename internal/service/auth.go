package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	authv1 "github.com/hyoureii/hrbackend/gen"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AuthServiceServer struct {
	db *gorm.DB
	authv1.UnimplementedAuthServiceServer
}

func NewAuthServiceServer(db *gorm.DB) *AuthServiceServer {
	return &AuthServiceServer{db: db}
}

func (srv AuthServiceServer) Register(c context.Context, r *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	hash, err := lib.HashPassword(r.Password)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
		FirstName: r.FirstName,
		LastName: r.LastName,
		Role: r.Role,
		AvatarURL: r.AvatarUrl,
		Email: r.Email,
		Password: string(hash),
	}
	err = gorm.G[models.User](srv.db).Create(c, newUser)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) { return nil, status.Error(codes.AlreadyExists, "User already exists")}
		return nil, err
	}

	return &authv1.RegisterResponse{}, nil
}

func (srv AuthServiceServer) Login(c context.Context, r *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	user, err := gorm.G[models.User](srv.db).Where("email = ?", r.Email).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, status.Error(codes.NotFound, "User not found")}
		return nil, err
	}
	ok := lib.ComparePassword(r.Password, []byte(user.Password))
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Incorrect Password")
	}

	return rotateRefreshToken(c, srv.db, user.ID, false)
}

func (srv AuthServiceServer) Refresh(c context.Context, r *authv1.RefreshRequest) (*authv1.LoginResponse, error) {
	token := r.RefreshToken
	claims, err := lib.ValidateJWT(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
	}
	userId, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, err
	}

	return rotateRefreshToken(c, srv.db, uint(userId), true)
}

func (srv AuthServiceServer) Logout(c context.Context, r *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.Claims)
	token, err := gorm.G[models.RefreshToken](srv.db).Where("user_id = ?", claims.Subject).Find(c)
	if err != nil {
		return nil, err
	}

	if len(token) != 0 {
		for _, row := range token {
			_, err := gorm.G[models.RefreshToken](srv.db).Where("id = ?", row.ID).Delete(c)
			if err != nil {
				return nil, err
			}
		}
	}
	_, err = gorm.G[models.RefreshToken](srv.db).Scopes(func(st *gorm.Statement) { st.Unscoped = true }).Where("deleted_at IS NOT NULL").Delete(c)
	if err != nil {
		return nil, err
	}

	return &authv1.LogoutResponse{}, nil
}

func (srv AuthServiceServer) Me(c context.Context, r *authv1.ProtectedRequest) (*authv1.Profile, error) {
	// TODO: implement
	return &authv1.Profile{}, nil
}

func rotateRefreshToken(c context.Context, db *gorm.DB, userId uint, refreshing bool) (*authv1.LoginResponse, error) {
	accessExp := time.Now().Add(time.Minute * 5)
	refreshExp := time.Now().Add(time.Hour * 24 * 7)

	refreshToken := lib.GenerateJWT(lib.ClaimRefresh, userId, refreshExp)
	hashStr := lib.HashToken(refreshToken)
	token, err := gorm.G[models.RefreshToken](db).Where("user_id = ?", userId).Find(c)
	if refreshing {
		latest := token[0]
		for _, row := range token {
			if row.ExpiredAt.After(latest.ExpiredAt) {
				latest = row
			}
		}
		if latest.ExpiredAt.Before(time.Now()) {
			return nil, status.Error(codes.Unauthenticated, "Refresh token expired")
		}
	}
	if err != nil {
		return nil, err
	}
	// NOTE: right now user can only have 1 refresh token on db,
	// maybe support for having multiple sessions(thus multiple refresh tokens)
	// will be added in the future
	if len(token) != 0 {
		for _, row := range token {
			_, err := gorm.G[models.RefreshToken](db).Where("id = ?", row.ID).Delete(c)
			if err != nil {
				return nil, err
			}
		}
	}
	_, err = gorm.G[models.RefreshToken](db).Scopes(func(st *gorm.Statement) { st.Unscoped = true }).Where("deleted_at IS NOT NULL").Delete(c)
	if err != nil {
		return nil, err
	}

	err = gorm.G[models.RefreshToken](db).Create(c, &models.RefreshToken{
		TokenHash: hashStr,
		ExpiredAt: refreshExp,
		UserID: userId,
	})
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken:  lib.GenerateJWT(lib.ClaimAccess, userId, accessExp),
		RefreshToken: refreshToken,
		ExpTime: accessExp.Unix(),
	}, nil
}
