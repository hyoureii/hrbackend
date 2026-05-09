package config

import (
	"fmt"

	"github.com/hyoureii/hrbackend/internal/lib"
)

type Config struct {
	GrpcPort        string
	HttpGatewayPort string
	DbDsn           string
}

func Load() *Config {
	dbDsn := buildDBDsn()
	return &Config{
		GrpcPort:        lib.GetEnvOrDefault("GRPC_PORT", "9000"),
		HttpGatewayPort: lib.GetEnvOrDefault("HTTP_GATEWAY_PORT", "9001"),
		DbDsn:           dbDsn,
	}
}

func buildDBDsn() string {
	host := lib.GetEnvOrDefault("POSTGRES_HOST", "localhost")
	user := lib.GetEnvOrDefault("POSTGRES_USER", "hrconnect")
	pw := lib.GetEnvOrDefault("POSTGRES_PASSWORD", "hrbackenddb")
	port := lib.GetEnvOrDefault("POSTGRES_PORT", "9002")

	dbUrl := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		pw,
		lib.GetEnvOrDefault("POSTGRES_DBNAME", "hr"),
		port,
	)

	return dbUrl
}
