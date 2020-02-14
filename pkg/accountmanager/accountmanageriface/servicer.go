//

package accountmanageriface

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"time"
)

// Servicer makes working with the Account Manager easier
//go:generate mockery -name Servicer
type Servicer interface {
	// ValidateAccess creates a new Account instance
	ValidateAccess(role *arn.ARN) error
	// Retrieve credentials for the provided IAM Role ARN
	Credentials(role *arn.ARN, roleSessionName string, duration *time.Duration) Credentialer
	// ConsoleURL generates a URL that may be used
	// to login to the AWS web console for an account
	ConsoleURL(creds Credentialer) (string, error)
	// UpsertPrincipalAccess creates roles, policies and update them as needed
	UpsertPrincipalAccess(account *account.Account) error
	// DeletePrincipalAccess removes all the principal roles and policies
	DeletePrincipalAccess(account *account.Account) error
}

//go:generate mockery -name Credentialer
type Credentialer interface {
	Get() (credentials.Value, error)
	ExpiresAt() (time.Time, error)
	IsExpired() bool
}
