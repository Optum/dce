package accountmanager

import (
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

//go:generate mockery -name clienter
type clienter interface {
	Config(roleArn *arn.ARN, roleSessionName string, duration *time.Duration) *aws.Config
	IAM(roleArn *arn.ARN) iamiface.IAMAPI
}

// Default IAM session name for operations performed by the DCE system
const systemSessionName = "DCESystem"

// Client helps with client management testing and abstraction
type client struct {
	session *session.Session
	sts     stsiface.STSAPI
}

// Config configures caching of credentials
func (c *client) Config(roleArn *arn.ARN, roleSessionName string, duration *time.Duration) *aws.Config {

	// return no config for nil inputs
	if roleArn == nil {
		return nil
	}

	// new creds
	creds := stscreds.NewCredentialsWithClient(c.sts, roleArn.String(), func(p *stscreds.AssumeRoleProvider) {
		if roleSessionName != "" {
			p.RoleSessionName = roleSessionName
		}
		if duration != nil {
			p.Duration = *duration
		}
	})

	// new config
	config := aws.NewConfig().WithCredentials(creds).WithMaxRetries(10)

	return config
}

// IAM creates a new IAM Client
func (c *client) IAM(roleArn *arn.ARN) iamiface.IAMAPI {
	return iam.New(c.session, c.Config(roleArn, systemSessionName, nil))
}
