package lib

import (
	"fmt"
	"log/slog"
	"os"
)

func GetEnv(env string) string {
	if foundEnv, found := os.LookupEnv(env); found {
		return foundEnv
	}
	return ""
}

func GetEnvOrDefault(env, def string) string {
	if foundEnv, found := os.LookupEnv(env); found {
		return foundEnv
	}
	slog.Warn(fmt.Sprintf("Environment variable %s isn't set. Using default %s", env, def))
	return def
}
