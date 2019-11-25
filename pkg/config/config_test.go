package config

import (
	"errors"
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
	ExpectedStrVal          = "foo"
	ExpectedStrDefaultedVal = "defaultfoo"
	ExpectedArrOfStrVal     = "one,two,three,four,five"
	ExpectedIntVal          = 1
)

func init() {
	os.Setenv("SOME_STRING_VALUE", ExpectedStrVal)
	os.Setenv("SOME_INT_VALUE", strconv.Itoa(ExpectedIntVal))
	os.Setenv("SOME_ARRAY_OF_STRING_VALUE", ExpectedArrOfStrVal)
}

type Foo interface {
	GetMessage() string
}

type Baz struct {
	Message string
}

func (b *Baz) GetMessage() string {
	return b.Message
}

type Fuzz interface {
	GetName() string
}

type Wuzz struct {
	Name string
}

func (w *Wuzz) GetName() string {
	return w.Name
}

func TestConfigBuilder_Unmarshal(t *testing.T) {
	actualStringVal := os.Getenv("SOME_STRING_VALUE")
	assert.Equal(t, ExpectedStrVal, actualStringVal)

	actualIntValAsStr := os.Getenv("SOME_INT_VALUE")
	actualIntVal, err := strconv.Atoi(actualIntValAsStr)
	assert.Nil(t, err)
	assert.Equal(t, ExpectedIntVal, actualIntVal)

	var config exampleConfig
	configurater := &ConfigurationBuilder{}

	err = configurater.Unmarshal(&config)

	assert.Nil(t, err)
	assert.Equal(t, ExpectedStrVal, config.StringValue)
	assert.Equal(t, ExpectedIntVal, config.IntValue)

	// Parsing an array
	expectedArrayOfStrings := []string{"one", "two", "three", "four", "five"}
	assert.Equal(t, expectedArrayOfStrings, config.ArrayOfStringValue)
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
	err := cfg.WithVal("bar", ExpectedStrVal).Build()
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, ExpectedStrVal, actualVal)
}

func TestConfigBuilder_BuildWithValueButGetWithError(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	err := cfg.WithVal("bar", ExpectedStrVal).Build()
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
	err := cfg.WithEnv("bar", "SOME_STRING_VALUE", ExpectedStrVal).Build()
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, ExpectedStrVal, actualVal)
}

func TestConfigBuilder_BuildWithEnvVarWithDefault(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	err := cfg.WithEnv("bar", "SOME_STRING_VALUE_THAT_DOES_NOT_EXIST", ExpectedStrDefaultedVal).Build()
	assert.Nil(t, err)

	actualVal, err := cfg.GetStringVal("bar")
	assert.Nil(t, err)
	assert.Equal(t, ExpectedStrDefaultedVal, actualVal)
}

func TestConfigBuilder_BuildWithService(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	baz := &Baz{Message: "Baz is the jazz!"}
	var iface Foo
	err := cfg.WithService(baz).Build()
	assert.Nil(t, err)

	err = cfg.GetService(&iface)
	assert.Nil(t, err)
	assert.Equal(t, "Baz is the jazz!", iface.GetMessage())
}

func TestConfigBuilder_BuildWithMulitpleServices(t *testing.T) {
	cfg := &ConfigurationBuilder{}
	var iface Foo
	var otherIface Fuzz

	baz := &Baz{Message: "Baz is the jazz!"}
	wuz := &Wuzz{Name: "The bear with no hair"}

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
	baz := &Baz{Message: "Baz is the jazz!"}
	var otherIface Fuzz

	expectedErr := ConfigurationError(errors.New("no service found in configuration for key type: config.Fuzz"))

	err := cfg.WithService(baz).Build()
	assert.Nil(t, err)

	err = cfg.GetService(&otherIface)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr.Error(), err.Error())
}
