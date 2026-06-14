package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"buf.build/go/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	useValidate "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/hyoureii/hrbackend/gen/attendance/v1"
	"github.com/hyoureii/hrbackend/gen/auth/v1"
	dashboard "github.com/hyoureii/hrbackend/gen/dashboard/v1"
	request "github.com/hyoureii/hrbackend/gen/request/v1"
	"github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/config"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/internal/service"
	"github.com/hyoureii/hrbackend/static"
	"github.com/redis/go-redis/v9"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Server struct {
	logger   *slog.Logger
	db       *gorm.DB
	rdb      *redis.Client
	grpcAddr string
	httpAddr string
}

func NewServer(logger *slog.Logger, conf *config.Config) (*Server, error) {
	db, err := gorm.Open(postgres.Open(conf.DbDsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		Username: conf.RedisUser,
	})
	if rdb == nil {
		return nil, err
	}

	return &Server{
		logger:   logger,
		db:       db,
		rdb:      rdb,
		grpcAddr: `:` + conf.GrpcPort,
		httpAddr: `:` + conf.HttpGatewayPort,
	}, nil
}

func (s *Server) Run(c context.Context, shutdownTimeout time.Duration) error {
	validator, err := protovalidate.New()
	if err != nil {
		return errors.Join(errors.New("Failed to create validator"), err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(interceptorLogger(s.logger)),
			middleware.UseValidateRequest(),
			middleware.UseAuth(),
			middleware.UseRBAC(),
			useValidate.UnaryServerInterceptor(validator),
		),
	)

	auth.RegisterAuthServiceServer(grpcServer, service.NewAuthServiceServer(s.db))
	users.RegisterUsersServiceServer(grpcServer, service.NewUsersServiceServer(s.db))
	attendance.RegisterAttendanceServiceServer(grpcServer, service.NewAttendanceServiceServer(s.db, s.rdb))
	request.RegisterLeaveServiceServer(grpcServer, service.NewLeaveServiceServer(s.db))
	request.RegisterTripServiceServer(grpcServer, service.NewTripServiceServer(s.db))
	dashboard.RegisterDashboardServiceServer(grpcServer, service.NewDashboardServiceServer(s.db))

	gatewayMux := runtime.NewServeMux()

	if err := registerGateway(c, gatewayMux, s.grpcAddr, auth.RegisterAuthServiceHandlerFromEndpoint); err != nil {
		return err
	}
	if err := registerGateway(c, gatewayMux, s.grpcAddr, users.RegisterUsersServiceHandlerFromEndpoint); err != nil {
		return err
	}
	if err := registerGateway(c, gatewayMux, s.grpcAddr, attendance.RegisterAttendanceServiceHandlerFromEndpoint); err != nil {
		return err
	}
	if err := registerGateway(c, gatewayMux, s.grpcAddr, request.RegisterLeaveServiceHandlerFromEndpoint); err != nil {
		return err
	}
	if err := registerGateway(c, gatewayMux, s.grpcAddr, request.RegisterTripServiceHandlerFromEndpoint); err != nil {
		return err
	}
	if err := registerGateway(c, gatewayMux, s.grpcAddr, dashboard.RegisterDashboardServiceHandlerFromEndpoint); err != nil {
		return err
	}

	handleStatic(gatewayMux, "/docs", "text/html", static.ScalarHtml)
	handleStatic(gatewayMux, "/scalar.js", "application/javascript", static.ScalarJS)
	handleStatic(gatewayMux, "/openapi.json", "application/json", static.OpenApiSpec)

	qrMux := http.NewServeMux()
	qrMux.HandleFunc("/qr/{id}", func(w http.ResponseWriter, r *http.Request) {
		png, err := s.rdb.Get(r.Context(), "qr:"+r.PathValue("id")).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				http.Error(w, "QR code not found or expired", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(png)
	})

	gateway := &http.Server{
		Addr: s.httpAddr,
		Handler: func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.logger.Info(fmt.Sprintf("[GATEWAY] Incoming %s %s", r.Method, r.RequestURI))
				body, _ := httputil.DumpRequest(r, true)
				s.logger.Debug(string(body))

				if after, ok := strings.CutPrefix(r.URL.Path, "/api/v1"); ok {
					r.URL.Path = after
					h.ServeHTTP(w, r)
				} else {
					qrMux.ServeHTTP(w, r)
					return
				}
			})
		}(gatewayMux),
	}

	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		return errors.Join(errors.New("Failed to create listener: %s"), err)
	}

	gsError := make(chan error, 1)
	go func() {
		s.logger.Info("Starting gRPC server...")
		if err := grpcServer.Serve(lis); !errors.Is(err, grpc.ErrServerStopped) {
			gsError <- err
		}
		close(gsError)
	}()
	s.logger.Info(fmt.Sprintf("Serving gRPC in %s", s.grpcAddr))

	gwError := make(chan error, 1)
	go func() {
		s.logger.Info("Starting REST gateway server...")
		if err := gateway.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			gwError <- err
		}
		close(gwError)
	}()
	s.logger.Info(fmt.Sprintf("Serving REST gateway in %s", gateway.Addr))

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-gsError:
		if err != nil {
			return err
		}
	case err := <-gwError:
		if err != nil {
			return err
		}
	case <-c.Done():
		s.logger.Info("Main context cancelled")
	case <-shutdown:
		s.logger.Info("Shutdown signal received, shutting down server gracefully")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err = gateway.Shutdown(shutdownCtx); err != nil {
		if closeErr := gateway.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	s.logger.Info("Server closed gracefully")
	return nil
}

func handleStatic(srv *runtime.ServeMux, path, contentType string, data []byte) {
	srv.HandlePath("GET", path, func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		w.Header().Set("Content-Type", contentType)
		_, err := w.Write(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
}

func registerGateway(
	ctx context.Context,
	r *runtime.ServeMux,
	address string,
	registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)) error {
	if err := registerFunc(
		ctx,
		r,
		address,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		return errors.Join(errors.New("Failed to register REST gateway: %s"), err)
	}
	return nil
}

func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}
