package lease

import (
	"errors"
	"reflect"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

// We don't use the internal errors package here because validation will rewrite it anyways
// Just spit out errors and turn them into validation errors inside the appropriate functions

var validateID = []validation.Rule{
	validation.NotNil.Error("must be a string"),
	is.UUIDv4.Error("must be a UUIDv4"),
}

var validateAccountID = []validation.Rule{
	validation.NotNil.Error("must be a string"),
	validation.Match(regexp.MustCompile("^[0-9]{12}$")).Error("must be a string with 12 digits"),
}

var validatePrincipalID = []validation.Rule{
	validation.NotNil.Error("must be a string"),
}

var validateInt64 = []validation.Rule{
	validation.NotNil.Error("must be an epoch timestamp"),
}

var validateStatus = []validation.Rule{
	validation.NotNil.Error("must be a valid lease status"),
}

func isNil(value interface{}) error {
	if !reflect.ValueOf(value).IsNil() {
		return errors.New("must be empty")
	}
	return nil
}

func isLeaseActive(value interface{}) error {
	s, _ := value.(*Status)
	if s.String() != StatusActive.String() {
		return errors.New("must be active lease")
	}
	return nil
}