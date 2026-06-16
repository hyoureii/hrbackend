package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hyoureii/hrbackend/gen/auth/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"github.com/redis/go-redis/v9"
)

type contextKey = string

const ClaimsKey contextKey = "claims"

var publicRoutes = map[string]bool{
	auth.AuthService_Login_FullMethodName:   true,
	auth.AuthService_Refresh_FullMethodName: true,
}

func UseAuth(rdb *redis.Client) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicRoutes[info.FullMethod] {
			return handler(ctx, req)
		}

		meta, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Missing metadata/headers")
		}

		accToken := meta.Get("authorization")
		if len(accToken) == 0 {
			return nil, status.Error(codes.Unauthenticated, "Missing authorization header")
		}

		tokenStr := strings.TrimPrefix(accToken[0], "Bearer ")

		claims, err := lib.ValidateJwt(tokenStr)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		hash := lib.HashToken(tokenStr)
		blacklisted, err := rdb.Exists(ctx, "blacklist:"+hash).Result()
		if err != nil {
			return nil, status.Error(codes.Internal, "Internal error")
		}
		if blacklisted > 0 {
			return nil, status.Error(codes.Unauthenticated, "Token revoked")
		}

		ctx = metadata.NewIncomingContext(context.WithValue(ctx, ClaimsKey, claims), meta)
		return handler(ctx, req)
	}
}
