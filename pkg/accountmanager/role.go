package accountmanager

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type createRoleInput struct {
	iam                      iamiface.IAMAPI
	roleArn                  arn.ARN
	description              *string
	maxSessionDuration       *int64
	document                 *string
	assumeRolePolicyDocument *string
	tags                     []*iam.Tag
}

// mergeRole - Create a Role, and attach a policy to it
func mergeRole(input *createRoleInput) error {
	_, err := input.iam.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(strings.Split(input.roleArn.String(), "/")[1]),
		AssumeRolePolicyDocument: input.assumeRolePolicyDocument,
		Description:              input.description,
		MaxSessionDuration:       input.maxSessionDuration,
		Tags:                     input.tags,
	})
	if err != nil {
		if isAWSAlreadyExistsError(err) {
			log.Print(err.Error() + " (Ignoring)")
		} else {
			return err
		}
	}

	return nil
}
