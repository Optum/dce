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

func isNilOrEqual(str string) validation.RuleFunc {
	return func(value interface{}) error {
		if !reflect.ValueOf(value).IsNil() {
			s, _ := value.(string)
			if s != str {
				return gErrors.New("unexpected string")
			}
		}
		return nil
	}
}

func isNilOrUsableAdminRole(am Manager) validation.RuleFunc {
	return func(value interface{}) error {
		fmt.Printf("Value: '%s'\n", value)
		if !reflect.ValueOf(value).IsNil() {
			s, _ := value.(string)
			return am.Setup(s)
		}
		return nil
	}
}
