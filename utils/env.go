package utils

import "os"

func GetEnv(env, def string) string {
	if foundEnv, found := os.LookupEnv(env); found {
		return foundEnv
	}
	return def
}
