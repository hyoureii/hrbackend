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

type ClaimScope string

const (
	ClaimAccess  ClaimScope = "access"
	ClaimRefresh ClaimScope = "refresh"
)

var jwtSecret []byte = []byte(GetEnv("JWT_SECRET"))

type Claims struct {
	jwt.RegisteredClaims
	Scope ClaimScope `json:"scope"`
	Perms []string   `json:"perms"`
	Role  string     `json:"role"`
}

type ClaimType string

const (
	CheckIn  ClaimType = "checkin"
	CheckOut ClaimType = "checkout"
)

type AttendanceClaims struct {
	jwt.RegisteredClaims
	Type ClaimType `json:"type"`
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateAccessRefresh(scope ClaimScope, userId string, role string, perms []string, exp time.Time) (string, error) {
	claims := &Claims{
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

func GenerateAttendanceJWT(typ ClaimType, exp time.Time) (string, error) {
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

func ValidateJwt(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("Invalid token")
	}
	return claims, nil
}
