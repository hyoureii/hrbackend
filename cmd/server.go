package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authv1 "github.com/hyoureii/hrbackend/gen/auth/v1"
	usersv1 "github.com/hyoureii/hrbackend/gen/users/v1"
	"github.com/hyoureii/hrbackend/internal/config"
	"github.com/hyoureii/hrbackend/internal/middleware"
	"github.com/hyoureii/hrbackend/internal/service"
	"github.com/hyoureii/hrbackend/static"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Server struct {
	db       *gorm.DB
	authDb *gorm.DB
	grpcAddr string
	httpAddr string
}

func NewServer(conf *config.Config) (*Server, error) {
	db, err := gorm.Open(postgres.Open(conf.DbDsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	authDb, err := gorm.Open(postgres.Open(conf.AuthDbDsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	log.Println("Database connection successful")

	return &Server{
		db:       db,
		authDb: authDb,
		grpcAddr: `:` + conf.GrpcPort,
		httpAddr: `:` + conf.HttpGatewayPort,
	}, nil
}

func (s *Server) Run() {
	ctx, shCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer shCancel()

	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		log.Fatalf("Failed to create listener: %s", err)
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(middleware.UseAuth()))
	authv1.RegisterAuthServiceServer(srv, service.NewAuthServiceServer(s.authDb))
	usersv1.RegisterUsersServiceServer(srv, service.NewUsersServiceServer(s.authDb))
	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %s", err)
		}
	}()
	log.Printf("Serving gRPC in %s", s.grpcAddr)

	gwMux := runtime.NewServeMux()

	registerGateway(ctx, gwMux, s.grpcAddr, authv1.RegisterAuthServiceHandlerFromEndpoint)
	registerGateway(ctx, gwMux, s.grpcAddr, usersv1.RegisterUsersServiceHandlerFromEndpoint)

	handleStatic(gwMux, "/docs", "text/html", static.ScalarHtml)
	handleStatic(gwMux, "/scalar.js", "application/javascript", static.ScalarJS)
	handleStatic(gwMux, "/openapi.json", "application/json", static.OpenApiSpec)

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1") {
			http.NotFound(w, r)
			return
		}
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/v1")
		gwMux.ServeHTTP(w, r)
	})

	logMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[GATEWAY] Incoming %s %s", r.Method, r.RequestURI)
		mux.ServeHTTP(w, r)
	})

	gateway := &http.Server{
		Addr:    s.httpAddr,
		Handler: logMux,
	}

	go func() {
		if err := gateway.ListenAndServe(); err != nil {
			log.Fatalf("Failed to serve REST gateway: %s", err)
		}
	}()
	log.Printf("Serving REST gateway in %s", gateway.Addr)

	<-ctx.Done()
	log.Println("Shutting down gracefully..")

	shCtx, shCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer shCancel()

	if err := gateway.Shutdown(shCtx); err != nil {
		log.Printf("Failed to gracefully shutdown REST gateway: %s", err)
	}
	srv.GracefulStop()
	log.Println("Server stopped")
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

func registerGateway(ctx context.Context, r *runtime.ServeMux, address string, registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)) {
	if err := registerFunc(
		ctx,
		r,
		address,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		log.Fatalf("Failed to register REST gateway: %s", err)
	}
}
