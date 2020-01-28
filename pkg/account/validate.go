package account

import (
	"errors"
	"reflect"
	"regexp"

	"github.com/Optum/dce/pkg/arn"
	validation "github.com/go-ozzo/ozzo-validation"
)

// We don't use the internal errors package here because validation will rewrite it anyways
// Just spit out errors and turn them into validation errors inside the appropriate functions

var validateAdminRoleArn = []validation.Rule{
	validation.NotNil.Error("must be a string"),
}

var validateID = []validation.Rule{
	validation.NotNil.Error("must be a string"),
	validation.Match(regexp.MustCompile("^[0-9]{12}$")).Error("must be a string with 12 digits"),
}

var validateInt64 = []validation.Rule{
	validation.NotNil.Error("must be an epoch timestamp"),
}

var validatePrincipalRoleArn = []validation.Rule{
	validation.NilOrNotEmpty.Error("must be an ARN or empty"),
}

var validatePrincipalPolicyHash = []validation.Rule{
	validation.NilOrNotEmpty.Error("must be a hash or empty"),
}

var validateStatus = []validation.Rule{
	validation.NotNil.Error("must be a valid account status"),
}

func isNil(value interface{}) error {
	if !reflect.ValueOf(value).IsNil() {
		return errors.New("must be empty")
	}
	return nil
}

func isNilOrUsableAdminRole(am Manager) validation.RuleFunc {
	return func(value interface{}) error {
		if !reflect.ValueOf(value).IsNil() {
			a, _ := value.(*arn.ARN)
			err := am.ValidateAccess(a)
			if err != nil {
				return errors.New("must be an admin role arn that can be assumed")
			}
		}
		return nil
	}
}

func isAccountNotLeased(value interface{}) error {
	s, _ := value.(*Status)
	if s.String() == StatusLeased.String() {
		return errors.New("must not be leased")
	}
	return nil
}
