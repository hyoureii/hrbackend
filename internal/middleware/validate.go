package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UseValidateRequest() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if req == nil {
			return nil, status.Errorf(codes.InvalidArgument, "nil request received for %s", info.FullMethod)
		}
		return handler(ctx, req)
	}
}
