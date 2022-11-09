package common

import (
	"os"
	"strconv"
	"strings"
)

// EnvConfig - Configuration pulled from the environment.
type EnvConfig interface {
	RequireEnvVar(varName string) string
	RequireEnvIntVar(varName string) int
	GetEnvVar(varName string, defaultValue string) string
	GetEnvIntVar(varName string, defaultValue int) int
	GetEnvFloatVar(varName string, defaultValue float64) float64
}

// DefaultEnvConfig - Default configuration
type DefaultEnvConfig struct{}

// RequireEnvVar - Requires the environment variable with the provided varName
//
//	and panics if the variable is not present.
func (cfg DefaultEnvConfig) RequireEnvVar(varName string) string {
	return RequireEnv(varName)
}

// RequireEnvIntVar - Requires the environment variable with the provided varName
//
//	and panics if the variable is not present.
func (cfg DefaultEnvConfig) RequireEnvIntVar(varName string) int {
	return RequireEnvInt(varName)
}

// GetEnvVar - Requires the environment variable with the provided varName
//
//	and panics if the variable is not present.
func (cfg DefaultEnvConfig) GetEnvVar(varName string, defaultValue string) string {
	return GetEnv(varName, defaultValue)
}

// GetEnvIntVar - Requires the environment variable with the provided varName
//
//	and panics if the variable is not present.
func (cfg DefaultEnvConfig) GetEnvIntVar(varName string, defaultValue int) int {
	return GetEnvInt(varName, defaultValue)
}

// GetEnvBoolVar - Requires the environment variable with the provided varName
//
//	and panics if the variable is not present.
func (cfg DefaultEnvConfig) GetEnvBoolVar(varName string, defaultValue bool) bool {
	strValue := GetEnv(varName, "false")
	return strings.ToLower(strValue) == "true"
}

// GetEnvFloatVar - Requires the environment variable with the provided varName
//
//	and panics if the variable is not present
func (cfg DefaultEnvConfig) GetEnvFloatVar(varName string, defaultValue float64) float64 {
	val, ok := os.LookupEnv(varName)

	if !ok {
		return defaultValue
	}

	floatVal, err := strconv.ParseFloat(val, 64)

	if err != nil {
		return defaultValue
	}

	return floatVal
}
