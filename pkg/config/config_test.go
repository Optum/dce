package config

import (
	"errors"
	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type exampleConfig struct {
	StringValue        string   `env:"SOME_STRING_VALUE"`
	ArrayOfStringValue []string `env:"SOME_ARRAY_OF_STRING_VALUE"`
	IntValue           int      `env:"SOME_INT_VALUE"`
}

const (
	ExpectedEnvStrVal       = "foo"
	ExpectedStrDefaultedVal = "defaultfoo"
	ExpectedArrOfStrVal     = "one,two,three,four,five"
	ExpectedIntVal          = 1
)

func init() {
	os.Setenv("SOME_STRING_VALUE", ExpectedEnvStrVal)
	os.Setenv("SOME_INT_VALUE", strconv.Itoa(ExpectedIntVal))
	os.Setenv("SOME_ARRAY_OF_STRING_VALUE", ExpectedArrOfStrVal)
}

type Messager interface {
	GetMessage() string
}

type MessageSvc struct {
	Message string
}

func (b *MessageSvc) GetMessage() string {
	return b.Message
}

type Namer interface {
	GetName() string
}

type NameSvc struct {
	Name string
}

func (w *NameSvc) GetName() string {
	return w.Name
}

func TestConfigBuilder_Unmarshal(t *testing.T) {
	actualStringVal := os.Getenv("SOME_STRING_VALUE")
	assert.Equal(t, ExpectedEnvStrVal, actualStringVal)

	actualIntValAsStr := os.Getenv("SOME_INT_VALUE")
	actualIntVal, err := strconv.Atoi(actualIntValAsStr)
	assert.Nil(t, err)
	assert.Equal(t, ExpectedIntVal, actualIntVal)

	var config exampleConfig
	configurater := &ConfigurationBuilder{}

	err = configurater.Unmarshal(&config)

	assert.Nil(t, err)
	assert.Equal(t, ExpectedEnvStrVal, config.StringValue)
	assert.Equal(t, ExpectedIntVal, config.IntValue)

	// Parsing an array
	expectedArrayOfStrings := []string{"one", "two", "three", "four", "five"}
	assert.Equal(t, expectedArrayOfStrings, config.ArrayOfStringValue)
}

func TestConfigBuilder_Dump(t *testing.T) {
	configurater := &ConfigurationBuilder{}
	configurater.WithVal("SOME_STRING_VALUE", ExpectedEnvStrVal)
	expectedArrayOfStrings := []string{"one", "two", "three", "four", "five"}
	configurater.WithVal("SOME_ARRAY_OF_STRING_VALUE", expectedArrayOfStrings)
	configurater.WithVal("SOME_INT_VALUE", ExpectedIntVal)

	var config exampleConfig

	err := configurater.Dump(&config)

	assert.Nil(t, err)
	assert.Equal(t, ExpectedEnvStrVal, config.StringValue)
	assert.Equal(t, ExpectedIntVal, config.IntValue)

	// Parsing an array
	assert.Equal(t, expectedArrayOfStrings, config.ArrayOfStringValue)
}

func TestConfigBuilder_DumpIntoSuperset(t *testing.T) {
	configurater := &ConfigurationBuilder{}
	configurater.WithVal("SOME_STRING_VALUE", ExpectedEnvStrVal)
	expectedArrayOfStrings := []string{"one", "two", "three", "four", "five"}
	configurater.WithVal("SOME_ARRAY_OF_STRING_VALUE", expectedArrayOfStrings)
	configurater.WithVal("SOME_INT_VALUE", ExpectedIntVal)

	var configSuperset struct {
		StringValue        string   `env:"SOME_STRING_VALUE"`
		ArrayOfStringValue []string `env:"SOME_ARRAY_OF_STRING_VALUE"`
		IntValue           int      `env:"SOME_INT_VALUE"`
		extraVal           string   //nolint
	}

	err := configurater.Dump(&configSuperset)
	assert.Nil(t, err)
	assert.Equal(t, ExpectedEnvStrVal, configSuperset.StringValue)
	assert.Equal(t, ExpectedIntVal, configSuperset.IntValue)

	// Parsing an array
	assert.Equal(t, expectedArrayOfStrings, configSuperset.ArrayOfStringValue)
}

func TestConfigBuilder_DumpIntoSubset(t *testing.T) {
	configurater := &ConfigurationBuilder{}
	configurater.WithVal("SOME_STRING_VALUE", ExpectedEnvStrVal)
	expectedArrayOfStrings := []string{"one", "two", "three", "four", "five"}
	configurater.WithVal("SOME_ARRAY_OF_STRING_VALUE", expectedArrayOfStrings)
	configurater.WithVal("SOME_INT_VALUE", ExpectedIntVal)

	var configSubset struct {
		StringValue        string   `env:"SOME_STRING_VALUE"`
		ArrayOfStringValue []string `env:"SOME_ARRAY_OF_STRING_VALUE"`
	}

	err := configurater.Dump(&configSubset)
	assert.Nil(t, err)
	assert.Equal(t, ExpectedEnvStrVal, configSubset.StringValue)

	// Parsing an array
	assert.Equal(t, expectedArrayOfStrings, configSubset.ArrayOfStringValue)
}

func TestConfigBuilder_TryToGetWithoutBuilding(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	cfg.WithVal("foo", "bar")

	actualVal, err := cfg.GetStringVal("foo")
	assert.NotNil(t, err)
	expectedErr := ConfigurationError(errors.New("call Build() before attempting to get values"))
	assert.Equal(t, expectedErr.Error(), err.Error())
	assert.True(t, len(actualVal) == 0)
}

func TestConfigBuilder_BuildWithValue(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	err := cfg.WithVal("bar", ExpectedEnvStrVal).Build()
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, ExpectedEnvStrVal, actualVal)
}

func TestConfigBuilder_BuildWithValueButGetWithError(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	err := cfg.WithVal("bar", ExpectedEnvStrVal).Build()
	// should have built just fine; error is in getting the key
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("somenonexistantkey")
	assert.NotNil(t, err)
	expectedErr := ConfigurationError(errors.New("no value found in configuration for key: somenonexistantkey"))
	assert.Equal(t, expectedErr.Error(), err.Error())
	assert.True(t, len(actualVal) == 0)
}

func TestConfigBuilder_BuildWithEnvVar(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	err := cfg.WithEnv("bar", "SOME_STRING_VALUE", ExpectedEnvStrVal).Build()
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, ExpectedEnvStrVal, actualVal)
}

func TestConfigBuilder_BuildWithEnvVarWithDefault(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	err := cfg.WithEnv("bar", "SOME_STRING_VALUE_THAT_DOES_NOT_EXIST", ExpectedStrDefaultedVal).Build()
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, ExpectedStrDefaultedVal, actualVal)
}

func TestConfigBuilder_BuildWithParameterStoreEnvVar(t *testing.T) {
	// Arrange
	var mockSSMClient = &mocks.SSMAPI{}
	svcBuilder := &ServiceBuilder{}
	svcBuilder.WithService(mockSSMClient)
	expectedSSMVal := "valueStoredInSSM"
	paramName := ExpectedEnvStrVal
	getParametersOutput := ssm.GetParametersOutput{
		Parameters: []*ssm.Parameter{
			&ssm.Parameter{
				Name:  &paramName,
				Value: &expectedSSMVal,
			},
		},
	}
	mockSSMClient.On("GetParameters", mock.MatchedBy(func(input *ssm.GetParametersInput) bool {
		return *input.Names[0] == ExpectedEnvStrVal && *input.WithDecryption == false
	})).Return(&getParametersOutput, nil)

	// Act
	svcBuilder.WithParameterStoreEnv("bar", "SOME_STRING_VALUE", "defaultVal")
	_, err := svcBuilder.Build()

	// Assert
	require.Nil(t, err)

	actualVal, err := svcBuilder.GetVal("bar")
	require.Nil(t, err)
	require.Equal(t, expectedSSMVal, actualVal)
}

func TestConfigBuilder_BuildWithParameterStoreEnvVar_UsesDefaultWhenInvalidParameter(t *testing.T) {
	// Arrange
	var mockSSMClient = &mocks.SSMAPI{}
	svcBuilder := &ServiceBuilder{}
	svcBuilder.WithService(mockSSMClient)
	paramName := ExpectedEnvStrVal
	getParametersOutput := ssm.GetParametersOutput{
		InvalidParameters: []*string{&paramName},
	}
	defaultValue := "defaultValue"
	mockSSMClient.On("GetParameters", mock.MatchedBy(func(input *ssm.GetParametersInput) bool {
		return *input.Names[0] == ExpectedEnvStrVal && *input.WithDecryption == false
	})).Return(&getParametersOutput, nil)

	// Act
	svcBuilder.WithParameterStoreEnv("bar", "SOME_STRING_VALUE", defaultValue)
	_, err := svcBuilder.Build()

	// Assert
	require.Nil(t, err)

	actualVal, err := svcBuilder.GetVal("bar")
	require.Nil(t, err)
	require.Equal(t, defaultValue, actualVal)
}

func TestConfigBuilder_BuildWithParameterStoreEnvVar_UsesDefaultWhenNoEnvVar(t *testing.T) {
	// Arrange
	cfg := &ConfigurationBuilder{}
	var mockSSMClient = &mocks.SSMAPI{}
	cfg.WithService(mockSSMClient)
	defaultValue := "defaultValue"
	mockSSMClient.AssertNotCalled(t, "GetParameters")

	// Act
	err := cfg.WithParameterStoreEnv("bar", "SOME_STRING_VALUE_THAT_DOESNT_EXIST", defaultValue).Build()

	// Assert
	assert.Nil(t, err)

	actualVal, err := cfg.GetVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, defaultValue, actualVal)
}

func TestConfigBuilder_BuildWithService(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	baz := &MessageSvc{Message: "Baz is the jazz!"}
	var iface Messager
	err := cfg.WithService(baz).Build()
	assert.Nil(t, err)

	err = cfg.GetService(&iface)
	assert.Nil(t, err)
	assert.Equal(t, "Baz is the jazz!", iface.GetMessage())
}

func TestConfigBuilder_BuildWithMulitpleServices(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	var iface Messager
	var otherIface Namer

	baz := &MessageSvc{Message: "Baz is the jazz!"}
	wuz := &NameSvc{Name: "The bear with no hair"}

	err := cfg.
		WithService(baz).
		WithService(wuz).
		Build()
	assert.Nil(t, err)

	err = cfg.GetService(&iface)
	assert.Nil(t, err)
	assert.Equal(t, "Baz is the jazz!", iface.GetMessage())

	err = cfg.GetService(&otherIface)
	assert.Nil(t, err)
	assert.Equal(t, "The bear with no hair", otherIface.GetName())
}

func TestConfigBuilder_BuildWithServiceWithError(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	baz := &MessageSvc{Message: "Baz is the jazz!"}
	var otherIface Namer

	expectedErr := ConfigurationError(errors.New("no service found in configuration for key type: config.Namer"))

	err := cfg.WithService(baz).Build()
	assert.Nil(t, err)

	err = cfg.GetService(&otherIface)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr.Error(), err.Error())
}

func TestConfigBuilder_UnmarshalService(t *testing.T) {
	type session struct {
		region string
	}
	type database struct {
		session   *session
		tableName string
	}
	type services struct {
		Foo     string `env:"FOO" envDefault:"bar"`
		Session *session
		Db      *database
	}

	sess := &session{region: "us-east-1"}
	db := &database{session: sess, tableName: "myTable"}

	// Setup config builder
	cfg := &ConfigurationBuilder{}
	cfg.
		WithService(sess).
		WithService(db)

	// Unmarshal into a service struct
	var svc services
	err := cfg.Unmarshal(&svc)
	require.Nil(t, err)
	require.NotNil(t, svc)

	require.Equal(t, "bar", svc.Foo)
	require.Equal(t, &session{region: "us-east-1"}, svc.Session)
	require.Equal(t, &database{session: sess, tableName: "myTable"}, svc.Db)
}

func TestConfigBuilder_UnmarshalValueIntoPointer(t *testing.T) {
	type database struct {
		tableName string
	}
	type services struct {
		// Load as pointer
		Db *database
	}

	cfg := &ConfigurationBuilder{}
	// Configure as value
	cfg.WithService(database{tableName: "myTable"})

	var svc services
	err := cfg.Unmarshal(&svc)
	require.Nil(t, err)
	require.NotNil(t, svc)

	require.NotNil(t, svc.Db)
	require.Equal(t, svc.Db, &database{tableName: "myTable"})
}

func TestConfigBuilder_UnmarshalPointerIntoValue(t *testing.T) {
	type database struct {
		tableName string
	}
	type services struct {
		// Load as value
		Db database
	}

	cfg := &ConfigurationBuilder{}
	// Configure as pointer
	cfg.WithService(&database{tableName: "myTable"})

	var svc services
	err := cfg.Unmarshal(&svc)
	require.Nil(t, err)
	require.NotNil(t, svc)

	require.NotNil(t, svc.Db)
	require.Equal(t, svc.Db, database{tableName: "myTable"})
}

type dber interface {
	Query()
}

// Implements dber
type database struct {
	tableName string
}

func (d *database) Query() {

}

func TestConfigBuilder_UnmarshalInterface(t *testing.T) {
	type services struct {
		Db dber
	}

	cfg := &ConfigurationBuilder{}
	cfg.WithService(&database{tableName: "myTable"})

	var svc services
	err := cfg.Unmarshal(&svc)
	require.Nil(t, err)
	require.NotNil(t, svc)

	require.NotNil(t, svc.Db)
	require.Equal(t, svc.Db, &database{tableName: "myTable"})
}
