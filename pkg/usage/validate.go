package usage

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
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
}

var validateTimeToLive = []validation.Rule{
	validation.NotNil.Error("must be a valid time to live"),
}
