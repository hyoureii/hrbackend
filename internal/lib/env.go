package lib

import "os"

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
	return def
}
