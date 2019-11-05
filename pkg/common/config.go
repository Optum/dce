package common

// EnvConfig - Configuration pulled from the environment.
type EnvConfig interface {
	RequireEnvVar(varName string) string
	RequireEnvVarInt(varName string) int
	GetEnvVar(varName string, defaultValue string) string
	GetEnvIntVar(varName string, defaultValue int) int
}

// DefaultEnvConfig - Default configuration
type DefaultEnvConfig struct{}

// RequireEnvVar - Requires the environment variable with the provided varName
//     and panics if the variable is not present.
func (cfg DefaultEnvConfig) RequireEnvVar(varName string) string {
	return RequireEnv(varName)
}

// RequireEnvIntVar - Requires the environment variable with the provided varName
//     and panics if the variable is not present.
func (cfg DefaultEnvConfig) RequireEnvIntVar(varName string) int {
	return RequireEnvInt(varName)
}

// GetEnvVar - Requires the environment variable with the provided varName
//     and panics if the variable is not present.
func (cfg DefaultEnvConfig) GetEnvVar(varName string, defaultValue string) string {
	return GetEnv(varName, defaultValue)
}

// GetEnvIntVar - Requires the environment variable with the provided varName
//     and panics if the variable is not present.
func (cfg DefaultEnvConfig) GetEnvIntVar(varName string, defaultValue int) int {
	return GetEnvInt(varName, defaultValue)
}
