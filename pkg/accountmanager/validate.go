package accountmanager

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	validation "github.com/go-ozzo/ozzo-validation"
)

func isArn(value interface{}) error {
	s, ok := value.(*string)
	if !ok {
		return fmt.Errorf("must be a string")
	}
	_, err := arn.Parse(*s)
	return err
}

func isAssumable(session *session.Session) validation.RuleFunc {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		// Create the credentials from AssumeRoleProvider to assume the role
		creds := stscreds.NewCredentials(session, s)
		_, err := creds.Get()
		if err != nil {
			return err
		}

		return nil
	}
}
