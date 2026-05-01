package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	pbAuth "github.com/hyoureii/hrbackend/gen"
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

	srv := grpc.NewServer(grpc.UnaryInterceptor(middleware.AuthUnaryInterceptor()))
	pbAuth.RegisterAuthServiceServer(srv, service.NewAuthServiceServer(s.authDb))
	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %s", err)
		}
	}()
	log.Printf("Serving gRPC in %s", s.grpcAddr)

	gwMux := runtime.NewServeMux()
	if err := pbAuth.RegisterAuthServiceHandlerFromEndpoint(
		ctx,
		gwMux,
		s.grpcAddr,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		log.Fatalf("Failed to register REST gateway: %s", err)
	}

	gwMux.HandlePath("GET", "/api/v1/docs", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write(static.ScalarHtml)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	gwMux.HandlePath("GET", "/api/v1/scalar.js", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		w.Header().Set("Content-Type", "application/javascript")
		_, err := w.Write(static.ScalarJS)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	gwMux.HandlePath("GET", "/api/gen/openapi.json", func(w http.ResponseWriter, r *http.Request, p map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(static.OpenApiSpec)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	gateway := &http.Server{
		Addr:    s.httpAddr,
		Handler: gwMux,
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
