//

package configiface

import "github.com/Optum/dce/pkg/config"

// ConfigurationBuilder makes working with the ConfigurationBuild easier
type ConfigurationBuilder interface {
	// Unmarshal loads configuration into the provided structure from environment variables.
	// Use the "env" tag on cfgStruct fields to indicate the corresponding environment variable to load from.
	Unmarshal(cfgStruct interface{}) error
	// Dump dumps the current config into the provided structure. Config keys are matched to
	// cfgStruct fields using the "env" tag.
	Dump(cfgStruct interface{}) error
	// WithService is a Builder Pattern method that allows you to specify services
	// for the given type.
	WithService(svc interface{}) *config.ConfigurationBuilder
	// WithEnv allows you to point to an environment variable for the value and
	// also specify a default using defaultValue
	WithEnv(key string, envVar string, defaultValue interface{}) *config.ConfigurationBuilder
	// WithParameterStoreEnv sets a config value from SSM Parameter store. The Parameter name is taken
	// from the provided environment variable. If the environment variable or SSM parameter can't be retrieved,
	// then the default value is used.
	// Requires that an SSM service of type ssmiface.SSMAPI is contained within config
	WithParameterStoreEnv(key string, envVar string, defaultValue string) *config.ConfigurationBuilder
	// WithVal allows you to hardcode string values into the configuration.
	// This is good for testing, injecting known values or values derived by means
	// outside the configuration.
	WithVal(key string, val interface{}) *config.ConfigurationBuilder
	// GetService retreives the service with the given type. An error is thrown if
	// the service is not found.
	GetService(svcFor interface{}) error
	// GetStringVal returns the value of the key as a string.
	GetStringVal(key string) (string, error)
	// GetVal returns the raw value
	GetVal(key string) (interface{}, error)
	// Build builds the configuration.
	Build() error
	// RetrieveParameterStoreVals - Get the values from the AWS Parameter Store
	RetrieveParameterStoreVals() error
}
