package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type ClaimType string

const (
	ClaimAccess  ClaimType = "access"
	ClaimRefresh ClaimType = "refresh"
)

var jwtSecret []byte = []byte(GetEnv("JWT_SECRET"))

type Claims struct {
	jwt.RegisteredClaims
	Scope ClaimType `json:"scope"`
}

func HashPassword(password string) ([]byte, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return []byte{}, err
	}
	return hashed, nil
}

func ComparePassword(password string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateJWT(scope ClaimType, userId string, exp time.Time) string {
	claims := &Claims{
		Scope: scope,
	}
	claims.Subject = fmt.Sprint(userId)
	claims.ExpiresAt = jwt.NewNumericDate(exp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to generate JWT token: %s", err))
	}
	return tokenStr
}

func ValidateJWT(tokenString string) (*Claims, error) {
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
