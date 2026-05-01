package config

import (
	"fmt"

	"github.com/hyoureii/hrbackend/utils"
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
		GrpcPort: utils.GetEnv("GRPC_PORT", "9000"),
		HttpGatewayPort: utils.GetEnv("HTTP_GATEWAY_PORT", "9001"),
		AuthDbDsn: authDbDsn,
		DbDsn: dbDsn,
	}
}

func buildDBDsn() (string, string) {
	host := utils.GetEnv("POSTGRES_HOST", "localhost")
	user := utils.GetEnv("POSTGRES_USER", "hrconnect")
	pw := utils.GetEnv("POSTGRES_PASSWORD", "hrbackenddb")
	port := utils.GetEnv("POSTGRES_PORT", "9002")

	authDbUrl := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		pw,
		utils.GetEnv("POSTGRES_AUTH_DBNAME", "auth"),
		port,
	)
	dbUrl := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		pw,
		utils.GetEnv("POSTGRES_AUTH_DBNAME", "hr"),
		port,
	)

	return authDbUrl, dbUrl
}
