package account

import (
	gErrors "errors"
	"fmt"
	"reflect"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
)

var accountIDRule = []validation.Rule{
	validation.Match(regexp.MustCompile("^[0-9]{12}$")).Error("must be a string with 12 digits"),
}

func isNilOrEqual(d interface{}) validation.RuleFunc {
	return func(value interface{}) error {
		if !reflect.ValueOf(value).IsNil() {
			s, _ := value.(*string)
			if *s != d {
				return gErrors.New("unexpected string")
			}
		}
		return nil
	}
}

func isNilOrUsableAdminRole(am Manager) validation.RuleFunc {
	return func(value interface{}) error {
		fmt.Printf("Here1\n")
		if !reflect.ValueOf(value).IsNil() {
			s, _ := value.(*string)
			fmt.Printf("Here2\n")
			return am.Setup(*s)
		}
		return nil
	}
}
