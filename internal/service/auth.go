package service

import (
	"context"
	"time"

	pb "github.com/hyoureii/hrbackend/gen"
	"github.com/hyoureii/hrbackend/utils"
)

type AuthServiceServer struct {
	pb.UnimplementedAuthServiceServer
}

func (AuthServiceServer) Register(c context.Context, r *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// TODO: implement
	return &pb.RegisterResponse{}, nil
}

func (AuthServiceServer) Login(c context.Context, r *pb.LoginRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{
		AccessToken: utils.GenerateJWT(),
		RefreshToken: utils.GenerateJWT(),
		ExpTime: time.Now().Add(5*time.Minute).Unix(),
	}, nil
}

func (AuthServiceServer) Refresh(c context.Context, r *pb.RefreshRequest) (*pb.LoginResponse, error) {
	// TODO: implement
	return &pb.LoginResponse{}, nil
}

func (AuthServiceServer) Logout(c context.Context, r *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	// TODO: implement
	return &pb.LogoutResponse{}, nil
}

func (AuthServiceServer) Me(c context.Context, r *pb.ProtectedRequest) (*pb.Profile, error) {
	// TODO: implement
	return &pb.Profile{}, nil
}
