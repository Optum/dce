package accountmanager

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
)

func isAWSAlreadyExistsError(err error) bool {
	aerr, ok := err.(awserr.Error)
	if ok {
		switch aerr.Code() {
		case iam.ErrCodeEntityAlreadyExistsException:
			return true
		}
	}

	return false
}

func iamResourceNameFromArn(arn arn.ARN) string {

	resourceName := strings.Split(arn.Resource, "/")

	return resourceName[len(resourceName)-1]
}
