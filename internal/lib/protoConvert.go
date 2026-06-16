package lib

import (
	"github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/models"
)

func UserDataToProto(l models.User) *users.User_Data {
	return &users.User_Data{
		Email:     l.Email,
		FirstName: l.FirstName,
		LastName:  l.LastName,
		Role:      l.Role.Name,
		AvatarUrl: l.AvatarURL,
	}
}
