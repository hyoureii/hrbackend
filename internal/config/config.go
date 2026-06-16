package config

import (
	"fmt"

	"github.com/hyoureii/hrbackend/internal/lib"
)

type S3Config struct {
	Addr      string
	SecretKey string
	AccessKey string
	Bucket    string
	Region    string
}

type Config struct {
	GrpcPort        string
	HttpGatewayPort string
	DbDsn           string
	RedisAddr       string
	RedisPass       string
	RedisUser       string
	S3              S3Config
}

func Load() *Config {
	dbDsn := buildDBDsn()
	return &Config{
		GrpcPort:        lib.GetEnvOrDefault("GRPC_PORT", "9000"),
		HttpGatewayPort: lib.GetEnvOrDefault("HTTP_GATEWAY_PORT", "9001"),
		DbDsn:           dbDsn,
		RedisAddr: fmt.Sprintf(
			"%s:%s",
			lib.GetEnvOrDefault("REDIS_HOST", "localhost"),
			lib.GetEnvOrDefault("REDIS_PORT", "6379"),
		),
		RedisPass: lib.GetEnvOrDefault("REDIS_PASSWORD", "hrconnect"),
		RedisUser: lib.GetEnvOrDefault("REDIS_USER", "hrbackendcache"),
		S3: S3Config{
			Addr: fmt.Sprintf(
				"http://%s:%s",
				lib.GetEnvOrDefault("GARAGE_S3_HOST", "localhost"),
				lib.GetEnvOrDefault("GARAGE_S3_PORT", "3900"),
			),
			SecretKey: lib.GetEnv("GARAGE_S3_SECRET"),
			AccessKey: lib.GetEnv("GARAGE_S3_ACCESS"),
			Bucket:    lib.GetEnvOrDefault("GARAGE_S3_BUCKET", "hrbackends3"),
			Region: lib.GetEnvOrDefault("GARAGE_S3_REGION", "hrconnect"),
		},
	}
}

func buildDBDsn() string {
	host := lib.GetEnvOrDefault("POSTGRES_HOST", "localhost")
	user := lib.GetEnvOrDefault("POSTGRES_USER", "hrconnect")
	pw := lib.GetEnvOrDefault("POSTGRES_PASSWORD", "hrbackenddb")
	port := lib.GetEnvOrDefault("POSTGRES_PORT", "5432")

	dbUrl := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host,
		user,
		pw,
		lib.GetEnvOrDefault("POSTGRES_DBNAME", "hr"),
		port,
	)

	return dbUrl
}
