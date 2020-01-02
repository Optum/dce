package account

import (
	"errors"
	"reflect"
	"regexp"

	"github.com/Optum/dce/pkg/model"
	validation "github.com/go-ozzo/ozzo-validation"
)

// We don't use the internal errors package here because validation will rewrite it anyways
// Just spit out errors and turn them into validation errors inside the appropriate functions

var accountIDRule = []validation.Rule{
	validation.Match(regexp.MustCompile("^[0-9]{12}$")).Error("must be a string with 12 digits"),
}

func isNil(value interface{}) error {
	if !reflect.ValueOf(value).IsNil() {
		return errors.New("should be nil")
	}
	return nil
}

func isNilOrEqual(d interface{}) validation.RuleFunc {
	return func(value interface{}) error {
		if !reflect.ValueOf(value).IsNil() {
			s, _ := value.(*string)
			if *s != d {
				return errors.New("is not nil or equal")
			}
		}
		return nil
	}
}

func isNilOrUsableAdminRole(am Manager) validation.RuleFunc {
	return func(value interface{}) error {
		if !reflect.ValueOf(value).IsNil() {
			s, _ := value.(*string)
			err := am.Setup(*s)
			if err != nil {
				return errors.New("cannot assume admin role arn")
			}
		}
		return nil
	}
}

func isAccountNotLeased(value interface{}) error {
	s, _ := value.(*model.AccountStatus)
	if *s == model.Leased {
		return errors.New("must not be leased")
	}
	return nil
}
