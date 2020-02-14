package accountmanager

import (
	"fmt"

	"github.com/Optum/dce/pkg/arn"
	validation "github.com/go-ozzo/ozzo-validation"
)

func isAssumable(client clienter) validation.RuleFunc {
	return func(value interface{}) error {
		a, ok := value.(*arn.ARN)
		if !ok {
			return fmt.Errorf("value is not an ARN")
		}
		// Create the credentials from AssumeRoleProvider to assume the role
		config := client.Config(a, systemSessionName, nil)
		_, err := config.Credentials.Get()
		if err != nil {
			return err
		}

		return nil
	}
}
