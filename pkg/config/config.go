package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env"
)

// ConfigurationError is an error that is returned by configuration
// methods when keys cannot be found or when there is an error whilst
// building the configuration.
type ConfigurationError error

// GenericConfiguration is a generic structure that contains configuration
type configurationValues struct {
	services  []interface{}
	types     []reflect.Type
	impls     []reflect.Value
	vals      map[string]interface{}
	errs      []error
	envKeys   map[string]string
	cfgStruct interface{}
}

// ConfigurationBuilder is the default implementation of a configuration loader.
type ConfigurationBuilder struct {
	values  *configurationValues
	parsers env.CustomParsers
	isBuilt bool
}

// Unmarshal loads the configuration into the provided structure.
func (config *ConfigurationBuilder) Unmarshal(cfgStruct interface{}) error {
	config.parsers = config.createCustomParsers()
	err := env.ParseWithFuncs(cfgStruct, config.parsers)
	return err
}

// WithService is a Builder Pattern method that allows you to specify services
// for the given type.
func (config *ConfigurationBuilder) WithService(svc interface{}) *ConfigurationBuilder {
	config.initialize()
	config.values.services = append(config.values.services, svc)
	config.values.types = append(config.values.types, reflect.TypeOf(svc))
	config.values.impls = append(config.values.impls, reflect.ValueOf(svc))
	return config
}

// WithEnv allows you to point to an environment variable for the value and
// also specify a default using defaultValue
func (config *ConfigurationBuilder) WithEnv(key string, envVar string, defaultValue interface{}) *ConfigurationBuilder {
	config.initialize()

	envVal, ok := os.LookupEnv(envVar)

	if !ok {
		config.values.vals[key] = defaultValue
	} else {
		config.values.vals[key] = envVal
	}

	return config
}

// WithVal allows you to hardcode string values into the configuration.
// This is good for testing, injecting known values or values derived by means
// outside the configuration.
func (config *ConfigurationBuilder) WithVal(key string, val interface{}) *ConfigurationBuilder {
	config.initialize()
	config.values.vals[key] = val
	return config
}

// GetService retreives the service with the given type. An error is thrown if
// the service is not found.
func (config *ConfigurationBuilder) GetService(svcFor interface{}) error {
	k := reflect.TypeOf(svcFor)
	kind := k.Kind()
	if kind == reflect.Ptr {
		k = k.Elem()
		kind = k.Kind()
	}
	for i, t := range config.values.types {
		if kind == reflect.Interface && t.Implements(k) {
			reflect.Indirect(
				reflect.ValueOf(svcFor),
			).Set(config.values.impls[i])
			return nil
		} else if kind == reflect.Struct && k.AssignableTo(t.Elem()) {
			reflect.ValueOf(svcFor).Elem().Set(config.values.impls[i].Elem())
			return nil
		}
	}
	return ConfigurationError(fmt.Errorf("no service found in configuration for key type: %s", k))
}

// GetStringVal returns the value of the key as a string.
func (config *ConfigurationBuilder) GetStringVal(key string) (string, error) {
	if !config.isBuilt {
		return "", ConfigurationError(errors.New("call Build() before attempting to get values"))
	}

	val, ok := config.values.vals[key]

	if !ok {
		return "", ConfigurationError(fmt.Errorf("no value found in configuration for key: %s", key))
	}

	return val.(string), nil
}

// GetBoolVal returns the environment variable as a boolean
func (config *ConfigurationBuilder) GetBoolVal(key string) (bool, error) {
	str, err := config.GetStringVal(key)
	return (strings.ToLower(str) == "true"), err
}

// GetVal returns the raw value
func (config *ConfigurationBuilder) GetVal(key string) (interface{}, error) {
	if !config.isBuilt {
		return "", ConfigurationError(errors.New("call Build() before attempting to get values"))
	}

	val, ok := config.values.vals[key]

	if !ok {
		return nil, ConfigurationError(fmt.Errorf("no value found in configuration for key: %s", key))
	}

	return val, nil
}

// Build builds the configuration.
func (config *ConfigurationBuilder) Build() error {
	// Add any "expensive" operations here. Validations, type conversions, etc.
	// We already have basic maps.
	config.isBuilt = true
	return nil
}

func (config *ConfigurationBuilder) initialize() {
	if config.values == nil {
		config.values = &configurationValues{}
	}
	if config.values.envKeys == nil {
		config.values.envKeys = make(map[string]string)
	}
	if config.values.vals == nil {
		config.values.vals = make(map[string]interface{})
	}
}

func (config *ConfigurationBuilder) createCustomParsers() env.CustomParsers {
	funcMap := env.CustomParsers{}
	return funcMap
}
