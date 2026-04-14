package utils

import (
	"crypto/rand"
	"encoding/base64"
	"log"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	jwt.RegisteredClaims
	DeviceID string `json="device_id"`
	Message string `json=message`
}

func NewClaims() Claims {
	claim := &Claims{}
	claim.DeviceID = "apalah"
	return *claim
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

func GenerateJWTSecret() string {
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatalf("Failed to generate jwt secret: %s\nPlease set JWT_SECRET in environment variable", err)
		}
		return base64.StdEncoding.EncodeToString(b)
}

func GenerateJWT() string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, NewClaims())
	tokenStr, err := token.SignedString([]byte(GetEnv("JWT_SECRET", GenerateJWTSecret())))
	if err != nil {
		log.Fatalf("Failed to generate JWT token: %s", err)
	}
	return tokenStr
}
