package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthClaimScope string

const (
	ClaimAccess  AuthClaimScope = "access"
	ClaimRefresh AuthClaimScope = "refresh"
)

var jwtSecret []byte = []byte(GetEnv("JWT_SECRET"))

type AuthClaims struct {
	jwt.RegisteredClaims
	Scope AuthClaimScope `json:"scope"`
	Perms []string   `json:"perms"`
	Role  string     `json:"role"`
}

type AttendanceClaimType string

const (
	CheckIn  AttendanceClaimType = "checkin"
	CheckOut AttendanceClaimType = "checkout"
)

type AttendanceClaims struct {
	jwt.RegisteredClaims
	Type AttendanceClaimType `json:"type"`
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateAccessRefresh(scope AuthClaimScope, userId string, role string, perms []string, exp time.Time) (string, error) {
	claims := &AuthClaims{
		Scope: scope,
		Perms: perms,
		Role:  role,
	}
	claims.Subject = userId
	claims.ExpiresAt = jwt.NewNumericDate(exp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to generate JWT token: %s", err))
		return "", err
	}
	return tokenStr, nil
}

func GenerateAttendanceJWT(typ AttendanceClaimType, exp time.Time) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	claims := &AttendanceClaims{
		Type: typ,
	}
	claims.ID = id.String()
	claims.ExpiresAt = jwt.NewNumericDate(exp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to generate JWT token: %s", err))
		return "", err
	}
	return tokenStr, nil
}

func ValidateJwt(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, errors.New("Invalid token")
	}
	return claims, nil
}
