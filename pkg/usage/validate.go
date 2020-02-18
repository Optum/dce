package usage

import (
	"errors"
	"reflect"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
)

var (
	validCurrencies = [...]string{
		"AUD",
		"CAD",
		"CHF",
		"CNY",
		"DKK",
		"EUR",
		"GBP",
		"HKD",
		"JPY",
		"NOK",
		"NZD",
		"SEK",
		"USD",
		"ZAR",
	}
)

// We don't use the internal errors package here because validation will rewrite it anyways
// Just spit out errors and turn them into validation errors inside the appropriate functions

var validatePrincipalID = []validation.Rule{
	validation.NotNil.Error("must be a valid principal ID"),
}

var validateAccountID = []validation.Rule{
	validation.NotNil.Error("must be a string"),
	validation.Match(regexp.MustCompile("^[0-9]{12}$")).Error("must be a string with 12 digits"),
}

var validateInt64 = []validation.Rule{
	validation.NotNil.Error("must be an epoch timestamp"),
}

var validateFloat64 = []validation.Rule{
	validation.NotNil.Error("must be a valid cost amount"),
}

var validateCostCurrency = []validation.Rule{
	validation.NotNil.Error("must be a valid cost concurrency"),
	validation.In(validCurrencies),
}

var validateTimeToLive = []validation.Rule{
	validation.NotNil.Error("must be a valid time to live"),
}

func isNil(value interface{}) error {
	if !reflect.ValueOf(value).IsNil() {
		return errors.New("must be empty")
	}
	return nil
}
