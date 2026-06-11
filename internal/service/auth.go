package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hyoureii/hrbackend/gen/auth/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpTime      int64
}

type AuthServiceServer struct {
	db *gorm.DB
	auth.UnimplementedAuthServiceServer
}

func NewAuthServiceServer(db *gorm.DB) *AuthServiceServer {
	return &AuthServiceServer{db: db}
}

func (s AuthServiceServer) Login(c context.Context, r *auth.LoginRequest) (*auth.LoginResponse, error) {
	valid, err := lib.ValidatePassword(r.Password)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "Password format invalid")
	}

	user, err := gorm.G[models.User](s.db).Joins(clause.LeftJoin.Association("Role"), nil).Where("email = ?", r.Email).First(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "User not found")
		}
		return nil, err
	}

	if !lib.ComparePassword(r.Password, []byte(user.Password)) {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password")
	}

	perms, err := getUserPermissions(c, s.db, user.RoleID)
	if err != nil {
		return nil, err
	}

	t, err := rotateRefreshToken(c, s.db, user.ID, user.Role.Name, perms, false)
	if err != nil {
		return nil, err
	}

	return &auth.LoginResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpTime:      t.ExpTime,
	}, nil
}

func (s AuthServiceServer) Refresh(c context.Context, r *auth.RefreshRequest) (*auth.RefreshResponse, error) {
	token := r.RefreshToken
	claims, err := lib.ValidateJwt(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
	}

	user, err := gorm.G[models.User](s.db).Joins(clause.LeftJoin.Association("Role"), nil).Where("users.id = ?", claims.Subject).First(c)
	if err != nil {
		return nil, err
	}

	perms, err := getUserPermissions(c, s.db, user.RoleID)
	if err != nil {
		return nil, err
	}

	t, err := rotateRefreshToken(c, s.db, claims.Subject, user.Role.Name, perms, true)
	if err != nil {
		return nil, err
	}
	return &auth.RefreshResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpTime:      t.ExpTime,
	}, nil
}

func (s AuthServiceServer) Logout(c context.Context, r *auth.LogoutRequest) (*auth.LogoutResponse, error) {
	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)
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

	return &auth.LogoutResponse{}, nil
}

func (s AuthServiceServer) ChangePassword(c context.Context, r *auth.ChangePasswordRequest) (*auth.ChangePasswordResponse, error) {
	valid, err := lib.ValidatePassword(r.OldPassword)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "Password format invalid")
	}
	valid, err = lib.ValidatePassword(r.NewPassword)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "Password format invalid")
	}

	claims := c.Value(middleware.ClaimsKey).(*lib.AuthClaims)
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

	return &auth.ChangePasswordResponse{}, nil
}

func (s AuthServiceServer) ResetPassword(c context.Context, r *auth.ResetPasswordRequest) (*auth.ResetPasswordResponse, error) {
	valid, err := lib.ValidatePassword(r.NewPassword)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "Password format invalid")
	}

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

	return &auth.ResetPasswordResponse{}, nil
}

func rotateRefreshToken(c context.Context, db *gorm.DB, userId string, role string, perms []string, refreshing bool) (*tokenResponse, error) {
	accessExp := time.Now().Add(time.Minute * 5)
	refreshExp := time.Now().Add(time.Hour * 24 * 7)

	refreshToken, err := lib.GenerateAccessRefresh(lib.ClaimRefresh, userId, role, perms, refreshExp)
	if err != nil {
		return nil, err
	}
	hashStr := lib.HashToken(refreshToken)
	rows, err := gorm.G[models.RefreshToken](db).Where("user_id = ?", userId).Find(c)
	if err != nil {
		return nil, err
	}
	if refreshing {
		if len(rows) <= 0 {
			return nil, status.Error(codes.Unauthenticated, "Refresh token invalid")
		}
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

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	err = gorm.G[models.RefreshToken](db).Create(c, &models.RefreshToken{
		Base:      models.Base{ID: id.String()},
		TokenHash: hashStr,
		ExpiredAt: refreshExp.Unix(),
		UserID:    userId,
	})
	if err != nil {
		return nil, err
	}
	accessToken, err := lib.GenerateAccessRefresh(lib.ClaimAccess, userId, role, perms, accessExp)
	if err != nil {
		return nil, err
	}

	return &tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpTime:      accessExp.Unix(),
	}, nil
}

func getUserPermissions(c context.Context, db *gorm.DB, roleID string) ([]string, error) {
	rolePerms, err := gorm.G[models.RolePermission](db).Joins(clause.LeftJoin.Association("Permission"), nil).Where("role_id = ?", roleID).Find(c)
	if err != nil {
		return nil, err
	}

	perms := make([]string, len(rolePerms))
	for i, rp := range rolePerms {
		perms[i] = rp.Permission.Codename
	}
	return perms, nil
}
