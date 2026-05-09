package service

import (
	"context"
	"errors"
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
	AccessToken  string
	RefreshToken string
	ExpTime      int64
}

type AuthServiceServer struct {
	db *gorm.DB
	authv1.UnimplementedAuthServiceServer
}

func NewAuthServiceServer(db *gorm.DB) *AuthServiceServer {
	return &AuthServiceServer{db: db}
}

func (s AuthServiceServer) Login(c context.Context, r *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	user, err := gorm.G[models.User](s.db).Where("email = ?", r.Email).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	if !lib.ComparePassword(r.Password, []byte(user.Password)) {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
	}

	t, err := rotateRefreshToken(c, s.db, user.ID, false)
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpTime:      t.ExpTime,
	}, nil
}

func (s AuthServiceServer) Refresh(c context.Context, r *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	token := r.RefreshToken
	claims, err := lib.ValidateJWT(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
	}

	t, err := rotateRefreshToken(c, s.db, claims.Subject, true)
	return &authv1.RefreshResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpTime:      t.ExpTime,
	}, nil
}

func (s AuthServiceServer) Logout(c context.Context, r *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.Claims)
	token, err := gorm.G[models.RefreshToken](s.db).Where("user_id = ?", claims.Subject).Find(c)
	if err != nil {
		return nil, err
	}

	if len(token) != 0 {
		for _, row := range token {
			_, err := gorm.G[models.RefreshToken](s.db).Where("id = ?", row.ID).Delete(c)
			if err != nil {
				return nil, err
			}
		}
	}

	return &authv1.LogoutResponse{}, nil
}

func (s AuthServiceServer) ChangePassword(c context.Context, r *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.Claims)
	user, err := gorm.G[models.User](s.db).Where("id = ?", claims.Subject).First(c)
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
	_, err = gorm.G[models.User](s.db).Where("id = ?", user.ID).Update(c, "password", hash)
	if err != nil {
		return nil, err
	}

	return &authv1.ChangePasswordResponse{}, nil
}

func (s AuthServiceServer) ResetPassword(c context.Context, r *authv1.ResetPasswordRequest) (*authv1.ResetPasswordResponse, error) {
	hash, err := lib.HashPassword(r.NewPassword)
	if err != nil {
		return nil, err
	}
	_, err = gorm.G[models.User](s.db).Where("id = ?", r.Id).Update(c, "password", hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	return &authv1.ResetPasswordResponse{}, nil
}

func rotateRefreshToken(c context.Context, db *gorm.DB, userId string, refreshing bool) (*tokenResponse, error) {
	accessExp := time.Now().Add(time.Minute * 5)
	refreshExp := time.Now().Add(time.Hour * 24 * 7)

	refreshToken := lib.GenerateJWT(lib.ClaimRefresh, userId, refreshExp)
	hashStr := lib.HashToken(refreshToken)
	rows, err := gorm.G[models.RefreshToken](db).Where("user_id = ?", userId).Find(c)
	if err != nil {
		return nil, err
	}
	if refreshing {
		latest := rows[0]
		for _, token := range rows {
			if time.Unix(token.ExpiredAt, 0).After(time.Unix(latest.ExpiredAt, 0)) {
				latest = token
			}
		}
		if time.Unix(latest.ExpiredAt, 0).Before(time.Now()) {
			return nil, status.Error(codes.Unauthenticated, "Refresh token expired")
		}
	}
	// NOTE: right now user can only have 1 refresh token on db,
	// maybe support for having multiple sessions(thus multiple refresh tokens)
	// will be added in the future
	if len(rows) != 0 {
		for _, row := range rows {
			_, err := gorm.G[models.RefreshToken](db).Where("id = ?", row.ID).Delete(c)
			if err != nil {
				return nil, err
			}
		}
	}

	err = gorm.G[models.RefreshToken](db).Create(c, &models.RefreshToken{
		TokenHash: hashStr,
		ExpiredAt: refreshExp.Unix(),
		UserID:    userId,
	})
	if err != nil {
		return nil, err
	}

	return &tokenResponse{
		AccessToken:  lib.GenerateJWT(lib.ClaimAccess, userId, accessExp),
		RefreshToken: refreshToken,
		ExpTime:      accessExp.Unix(),
	}, nil
}
