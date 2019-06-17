package common

import (
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/sts"
)

// TokenService interface requires a method to receive credentials for an AWS
// Role provided by the Role Input.
type TokenService interface {
	AssumeRole(*sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
	NewCredentials(client.ConfigProvider, string) *credentials.Credentials
}

// STS implements the TokenService interface using AWS STS Client
type STS struct {
	Client *sts.STS
}

// AssumeRole returns an STS AssumeRoleOutput struct based on the provided
// input through the AWS STS Client
func (service STS) AssumeRole(input *sts.AssumeRoleInput) (
	*sts.AssumeRoleOutput, error) {
	return service.Client.AssumeRole(input)
}

// NewCredentials returns a set of credentials for an Assume Role
func (service STS) NewCredentials(inputClient client.ConfigProvider,
	inputRole string) *credentials.Credentials {
	return stscreds.NewCredentials(inputClient, inputRole)
}
