package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hyoureii/hrbackend/internal/lib"
)

type contextKey = string

const ClaimsKey contextKey = "claims"

var publicRoutes = map[string]bool{
	"/auth.v1.AuthService/Login":   true,
	"/auth.v1.AuthService/Refresh": true,
}

func UseAuth() grpc.UnaryServerInterceptor {
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

		claims, err := lib.ValidateJWT(tokenStr)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "Invalid token")
		}

		ctx = metadata.NewIncomingContext(context.WithValue(ctx, ClaimsKey, claims), meta)
		return handler(ctx, req)
	}
}
