package usage

import (
	validation "github.com/go-ozzo/ozzo-validation"
)

var (
	validCurrencies = []interface{}{
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

var validationLeaseID = []validation.Rule{
	validation.NotNil.Error("must be a valid lease ID"),
}

var validateInt64 = []validation.Rule{
	validation.NotNil.Error("must be an epoch timestamp"),
}

var validateFloat64 = []validation.Rule{
	validation.NotNil.Error("must be a valid cost amount"),
}

var validateCostCurrency = []validation.Rule{
	validation.NotNil.Error("must be a valid cost currency"),
	validation.In(validCurrencies...),
}

var validateTimeToLive = []validation.Rule{
	validation.NotNil.Error("must be a valid time to live"),
}
