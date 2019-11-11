package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// RequireEnv validates and returns an environment variable's value if it exists
func RequireEnv(env string) string {
	val, ok := os.LookupEnv(env)
	if !ok {
		panic(fmt.Errorf("missing required env var %v", env))
	}
	return val
}

// RequireEnvInt returns an environment that is required to be an integer
func RequireEnvInt(env string) int {
	val := RequireEnv(env)
	intVal, err := strconv.Atoi(val)

	if err != nil {
		panic(fmt.Errorf("unable to parse env var %s as int: %s", env, val))
	}

	return intVal
}

// RequireEnvFloat returns an environment that is required to be an float64
func RequireEnvFloat(env string) float64 {
	val := RequireEnv(env)
	floatVal, err := strconv.ParseFloat(val, 64)

	if err != nil {
		panic(fmt.Errorf("unable to parse env var %s as float: %s", env, val))
	}

	return floatVal
}

// GetEnv returns an environment. The defaultValue is returned if the variable does not exist.
func GetEnv(env string, defaultValue string) string {
	val, ok := os.LookupEnv(env)

	if !ok || len(env) == 0 {
		return defaultValue
	}

	return val
}

// GetEnvInt returns an environment that is required to be an integer
func GetEnvInt(env string, defaultValue int) int {
	val, ok := os.LookupEnv(env)

	if !ok {
		return defaultValue
	}

	intVal, err := strconv.Atoi(val)

	if err != nil {
		return defaultValue
	}

	return intVal
}

// RequireEnvStringSlice - Requires the given environment variable to contain a slice of string
func RequireEnvStringSlice(env string, sep string) []string {
	val := RequireEnv(env)
	list := strings.Split(val, sep)

	// Remove empty values
	cleanList := []string{}
	for _, value := range list {
		if value != "" {
			cleanList = append(cleanList, value)
		}
	}

	return cleanList
}

// RequireEnvFloatSlice - Requires the given environment variable to contain a slice of float64
func RequireEnvFloatSlice(env string, sep string) []float64 {
	strList := RequireEnvStringSlice(env, sep)

	floatList := []float64{}
	for _, val := range strList {
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			panic(fmt.Errorf("env var %s contains invalid float64 value: %s", env, val))
		}
		floatList = append(floatList, floatVal)
	}

	return floatList
}
