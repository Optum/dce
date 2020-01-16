package lease

import (
	"errors"
	"reflect"
	"regexp"

	"github.com/Optum/dce/pkg/model"
	validation "github.com/go-ozzo/ozzo-validation"
)

// We don't use the internal errors package here because validation will rewrite it anyways
// Just spit out errors and turn them into validation errors inside the appropriate functions

var validateID = []validation.Rule{
	validation.NotNil.Error("must be a string"),
	validation.Match(regexp.MustCompile("^[0-9]{12}$")).Error("must be a string with 12 digits"),
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

func isLeaseNotActive(value interface{}) error {
	s, _ := value.(*model.LeaseStatus)
	if s.String() == model.LeaseStatusActive.String() {
		return errors.New("must not be active")
	}
	return nil
}
