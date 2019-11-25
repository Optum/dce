package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/caarlos0/env"
)

// ConfigurationError is an error that is returned by configuration
// methods when keys cannot be found or when there is an error whilst
// building the configuration.
type ConfigurationError error

// GenericConfiguration is a generic structure that contains configuration
type genericConfiguration struct {
	services  []interface{}
	types     []reflect.Type
	impls     []reflect.Value
	vals      map[string]interface{}
	errs      []error
	envKeys   map[string]string
	cfgStruct interface{}
}

// DCEConfigBuilder is the default implementation of a configuration loader.
type DCEConfigBuilder struct {
	values    *genericConfiguration
	parsers   env.CustomParsers
	isBuilt   bool
	useStruct bool
}

// WithStruct loads the configuration into the provided structure.
func (config *DCEConfigBuilder) WithStruct(cfgStruct interface{}) *DCEConfigBuilder {
	config.useStruct = true
	config.initialize()
	config.values.cfgStruct = cfgStruct
	return config
}

// WithService is a Builder Pattern method that allows you to specify services
// for the given type.
func (config *DCEConfigBuilder) WithService(svc interface{}) *DCEConfigBuilder {
	config.initialize()
	config.values.services = append(config.values.services, svc)
	config.values.types = append(config.values.types, reflect.TypeOf(svc))
	config.values.impls = append(config.values.impls, reflect.ValueOf(svc))
	return config
}

// WithEnv allows you to point to an environment variable for the value and
// also specify a default using defaultValu
func (config *DCEConfigBuilder) WithEnv(key string, envVar string, defaultValue interface{}) *DCEConfigBuilder {
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
func (config *DCEConfigBuilder) WithVal(key string, val interface{}) *DCEConfigBuilder {
	config.initialize()
	config.values.vals[key] = val
	return config
}

// GetService retreives the service with the given type. An error is thrown if
// the service is not found.
func (config *DCEConfigBuilder) GetService(svcFor interface{}) error {
	k := reflect.TypeOf(svcFor).Elem()
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
			reflect.ValueOf(svcFor).Elem().Set(config.values.impls[i])
			return nil
		}
	}
	return ConfigurationError(fmt.Errorf("no service found in configuration for key type: %s", k))
}

// GetStruct returns the populated struct specified in the `WithStruct` method.
func (config *DCEConfigBuilder) GetStruct() (interface{}, error) {
	if !config.useStruct {
		// TODO: Not sure if this should throw an error or just an empty struct...
		return nil, ConfigurationError(errors.New("call `WithStruct` to supply a struct first"))
	}

	if !config.isBuilt {
		return nil, ConfigurationError(errors.New("call Build() before attempting to get values"))
	}
	return config.values.cfgStruct, nil
}

// GetStringVal returns the value of the key as a string.
func (config *DCEConfigBuilder) GetStringVal(key string) (string, error) {
	if !config.isBuilt {
		return "", ConfigurationError(errors.New("call Build() before attempting to get values"))
	}

	val, ok := config.values.vals[key]

	if !ok {
		return "", ConfigurationError(fmt.Errorf("no value found in configuration for key: %s", key))
	}

	return val.(string), nil
}

// GetVal returns the raw value
func (config *DCEConfigBuilder) GetVal(key string) (interface{}, error) {
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
func (config *DCEConfigBuilder) Build() error {
	if config.useStruct {
		err := config.buildWithStruct()
		if err != nil {
			return ConfigurationError(fmt.Errorf("error while trying to parse configuration object: %s", err.Error()))
		}
	}
	// TODO: Add any "expensive" operations here. Validations, type conversions, etc.
	// We already have basic maps.
	config.isBuilt = true
	return nil
}

func (config *DCEConfigBuilder) buildWithStruct() error {
	config.parsers = config.createCustomParsers()
	err := env.ParseWithFuncs(config.values.cfgStruct, config.parsers)
	// TODO: Add some more context to the error and consider wrapping with
	// our ConfigurationError
	return err
}

func (config *DCEConfigBuilder) initialize() {
	if config.values == nil {
		config.values = &genericConfiguration{}
	}
	if config.values.envKeys == nil {
		config.values.envKeys = make(map[string]string)
	}
	if config.values.vals == nil {
		config.values.vals = make(map[string]interface{})
	}
}

func (config *DCEConfigBuilder) createCustomParsers() env.CustomParsers {
	funcMap := env.CustomParsers{}
	return funcMap
}
