package common

import (
	"fmt"
	"os"
)

// RequireEnv validates and returns an environment variable's value if it exists
func RequireEnv(env string) string {
	val, ok := os.LookupEnv(env)
	if !ok {
		panic(fmt.Errorf("missing required env var %v", env))
	}
	return val
}
