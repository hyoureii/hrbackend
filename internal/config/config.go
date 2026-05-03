package config

import (
	"fmt"

	"github.com/hyoureii/hrbackend/internal/lib"
)

type Config struct {
	GrpcPort string
	HttpGatewayPort string
	AuthDbDsn string
	DbDsn string
}

func Load() *Config {
	authDbDsn, dbDsn := buildDBDsn()
	return &Config{
		GrpcPort: lib.GetEnv("GRPC_PORT", "9000"),
		HttpGatewayPort: lib.GetEnv("HTTP_GATEWAY_PORT", "9001"),
		AuthDbDsn: authDbDsn,
		DbDsn: dbDsn,
	}
}

func buildDBDsn() (string, string) {
	host := lib.GetEnv("POSTGRES_HOST", "localhost")
	user := lib.GetEnv("POSTGRES_USER", "hrconnect")
	pw := lib.GetEnv("POSTGRES_PASSWORD", "hrbackenddb")
	port := lib.GetEnv("POSTGRES_PORT", "9002")

	authDbUrl := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		pw,
		lib.GetEnv("POSTGRES_AUTH_DBNAME", "auth"),
		port,
	)
	dbUrl := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		pw,
		lib.GetEnv("POSTGRES_AUTH_DBNAME", "hr"),
		port,
	)

	return authDbUrl, dbUrl
}
