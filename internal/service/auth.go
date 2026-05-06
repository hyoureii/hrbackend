package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	authv1 "github.com/hyoureii/hrbackend/gen/auth/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type tokenResponse struct {
	AccessToken string
	RefreshToken string
	ExpTime int64
}

type AuthServiceServer struct {
	db *gorm.DB
	authv1.UnimplementedAuthServiceServer
}

func NewAuthServiceServer(db *gorm.DB) *AuthServiceServer {
	return &AuthServiceServer{db: db}
}

func (srv AuthServiceServer) Login(c context.Context, r *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	user, err := gorm.G[models.User](srv.db).Where("email = ?", r.Email).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, status.Error(codes.NotFound, "User not found")}
		return nil, err
	}
	
	if !lib.ComparePassword(r.Password, []byte(user.Password)) {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
	}

	t, err := rotateRefreshToken(c, srv.db, user.ID, false)
	if err != nil { return nil, err }

	return &authv1.LoginResponse{
		AccessToken: t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpTime: t.ExpTime,
	}, nil
}

func (srv AuthServiceServer) Refresh(c context.Context, r *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	token := r.RefreshToken
	claims, err := lib.ValidateJWT(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
	}
	userId, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, err
	}

	t, err := rotateRefreshToken(c, srv.db, uint(userId), true)
	return &authv1.RefreshResponse{
		AccessToken: t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpTime: t.ExpTime,
	}, nil
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

func (srv AuthServiceServer) ChangePassword(c context.Context, r *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.Claims)
	user, err := gorm.G[models.User](srv.db).Where("id = ?", claims.Subject).First(c)
	if err != nil {
		return nil, err
	}

	if !lib.ComparePassword(r.OldPassword, []byte(user.Password)) {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
	}

	hash, err := lib.HashPassword(r.NewPassword)
	if err != nil {
		return nil, err
	}
	_, err = gorm.G[models.User](srv.db).Where("id = ?", user.ID).Update(c, "password", hash)
	if err != nil {
		return nil, err
	}

	return &authv1.ChangePasswordResponse{}, nil
}

func (srv AuthServiceServer) ResetPassword(c context.Context, r *authv1.ResetPasswordRequest) (*authv1.ResetPasswordResponse, error) {
	user, err := gorm.G[models.User](srv.db).Where("id = ?", r.Id).First(c)
	if err != nil {
		return nil, err
	}

	hash, err := lib.HashPassword(r.NewPassword)
	if err != nil {
		return nil, err
	}
	_, err = gorm.G[models.User](srv.db).Where("id = ?", user.ID).Update(c, "password", hash)
	if err != nil {
		return nil, err
	}

	return &authv1.ResetPasswordResponse{}, nil
}

func rotateRefreshToken(c context.Context, db *gorm.DB, userId uint, refreshing bool) (*tokenResponse, error) {
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

	return &tokenResponse{
		AccessToken:  lib.GenerateJWT(lib.ClaimAccess, userId, accessExp),
		RefreshToken: refreshToken,
		ExpTime: accessExp.Unix(),
	}, nil
}
