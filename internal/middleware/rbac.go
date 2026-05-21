package middleware

import (
	"context"

	attendance "github.com/hyoureii/hrbackend/gen/attendance/v1"
	request "github.com/hyoureii/hrbackend/gen/request/v1"
	users "github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var protectedEndpoints = map[string]string{
	users.UsersService_Register_FullMethodName:              "manageUsers",
	users.UsersService_Update_FullMethodName:                "manageUsers",
	users.UsersService_Delete_FullMethodName:                "manageUsers",
	users.UsersService_Activate_FullMethodName:              "manageUsers",
	users.UsersService_Deactivate_FullMethodName:            "manageUsers",
	attendance.AttendanceService_Generate_FullMethodName:    "createAttendanceQR",
	attendance.AttendanceService_Today_FullMethodName:       "createAttendanceQR",
	request.LeaveService_UpdateLeave_FullMethodName:         "manageLeaveRequest",
	request.LeaveService_ApproveLeave_FullMethodName:        "manageLeaveRequest",
	request.LeaveService_RejectLeave_FullMethodName:         "manageLeaveRequest",
	request.LeaveService_GetAllPendingLeaves_FullMethodName: "manageLeaveRequest",
	request.TripService_UpdateTrip_FullMethodName:           "manageTripRequest",
	request.TripService_ApproveTrip_FullMethodName:          "manageTripRequest",
	request.TripService_RejectTrip_FullMethodName:           "manageTripRequest",
	request.TripService_GetAllPendingTrips_FullMethodName:   "manageTripRequest",
}

func UseRBAC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		perm, ok := protectedEndpoints[info.FullMethod]
		if !ok {
			return handler(ctx, req)
		}

		claims, ok := ctx.Value(ClaimsKey).(*lib.Claims)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Missing claims")
		}
		if !lib.HasPermission(claims.Perms, perm) {
			return nil, status.Error(codes.PermissionDenied, "You do not have permission to access this endpoint")
		}

		return handler(ctx, req)
	}
}
