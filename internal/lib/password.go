package lib

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

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

func ValidatePassword(password string) (bool, error) {
	regex := []string{
		"[a-z]",
		"[A-Z]",
		"[0-9]",
		"[!@#$%^&*(),.?\\\":{}|<>]",
	}

	for _, pattern := range regex {
		patternRegex, err := regexp.Compile(pattern)
		if err != nil {
			return false, err
		}
		match := patternRegex.MatchString(password)
		if !match {
			return false, err
		}
	}
	return true, nil
}
