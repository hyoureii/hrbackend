package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
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
	grpcAddr string
	httpAddr string
}

func NewServer(conf *config.Config) (*Server, error) {
	db, err := gorm.Open(postgres.Open(conf.DbDsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	log.Println("Database connection successful")

	return &Server{
		db:       db,
		grpcAddr: `:` + conf.GrpcPort,
		httpAddr: `:` + conf.HttpGatewayPort,
	}, nil
}

func (s *Server) Run(c context.Context, shutdownTimeout time.Duration) error {

	gs := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.UseAuth()),
	)

	authv1.RegisterAuthServiceServer(gs, service.NewAuthServiceServer(s.db))
	usersv1.RegisterUsersServiceServer(gs, service.NewUsersServiceServer(s.db))

	gwMux := runtime.NewServeMux()

	registerGateway(c, gwMux, s.grpcAddr, authv1.RegisterAuthServiceHandlerFromEndpoint)
	registerGateway(c, gwMux, s.grpcAddr, usersv1.RegisterUsersServiceHandlerFromEndpoint)

	handleStatic(gwMux, "/docs", "text/html", static.ScalarHtml)
	handleStatic(gwMux, "/scalar.js", "application/javascript", static.ScalarJS)
	handleStatic(gwMux, "/openapi.json", "application/json", static.OpenApiSpec)

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[GATEWAY] Incoming %s %s", r.Method, r.RequestURI)

		if after, ok := strings.CutPrefix(r.URL.Path, "/api/v1"); ok {
			r.URL.Path = after
			gwMux.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
			return
		}
	})

	gw := &http.Server{
		Addr:    s.httpAddr,
		Handler: mux,
	}

	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		log.Fatalf("Failed to create listener: %s", err)
	}

	gsError := make(chan error, 1)
	go func() {
		log.Println("Starting gRPC server...")
		if err := gs.Serve(lis); !errors.Is(err, grpc.ErrServerStopped) {
			gsError <- err
		}
		close(gsError)
	}()
	log.Printf("Serving gRPC in %s", s.grpcAddr)

	gwError := make(chan error, 1)
	go func() {
		log.Println("Starting REST gateway server...")
		if err := gw.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			gwError <- err
		}
		close(gwError)
	}()
	log.Printf("Serving REST gateway in %s", gw.Addr)

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
		log.Println("Main context cancelled")
	case <-shutdown:
		log.Println("Shutdown signal received, shutting down server gracefully")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err = gw.Shutdown(shutdownCtx); err != nil {
		if closeErr := gw.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	log.Println("Server closed gracefully")
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
	registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error)) {
	if err := registerFunc(
		ctx,
		r,
		address,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		log.Fatalf("Failed to register REST gateway: %s", err)
	}
}
