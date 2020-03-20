package accountmanager

import (
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

type clienter interface {
	Config(roleArn *arn.ARN) *aws.Config
	IAM(roleArn *arn.ARN) iamiface.IAMAPI
}

// Client helps with client management testing and abstraction
type client struct {
	session *session.Session
	sts     stsiface.STSAPI
	configs map[string]*aws.Config
}

// Config configures caching of credentials
func (c *client) Config(roleArn *arn.ARN) *aws.Config {

	// return no config for nil inputs
	if roleArn == nil {
		return nil
	}

	key := roleArn.String()

	// check for cached config
	if c.configs != nil && c.configs[key] != nil {
		return c.configs[key]
	}

	// new creds
	creds := stscreds.NewCredentialsWithClient(c.sts, roleArn.String())

	// new config
	config := aws.NewConfig().WithCredentials(creds).WithMaxRetries(10)

	if c.configs == nil {
		c.configs = map[string]*aws.Config{}
	}

	c.configs[key] = config
	return config
}

// IAM creates a new IAM Client
func (c *client) IAM(roleArn *arn.ARN) iamiface.IAMAPI {
	return iam.New(c.session, c.Config(roleArn))
}
