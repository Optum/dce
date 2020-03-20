package arn

import (
	"strconv"
	"strings"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// ARN - Custom ARN type for helping with formatting
type ARN struct {
	arn.ARN
}

// UnmarshalJSON - Custom unmarshalling of an ARN
func (a *ARN) UnmarshalJSON(data []byte) error {

	unquoted, err := strconv.Unquote(string(data))
	if err != nil {
		return errors.NewInternalServer("unexpected error unqouting string", err)
	}
	err = a.parseString(unquoted)
	if err != nil {
		return err
	}
	return nil
}

// MarshalJSON for creating a JSON string of ARN
func (a *ARN) MarshalJSON() ([]byte, error) {
	quoted := strconv.Quote(a.String())
	return []byte(quoted), nil
}

// UnmarshalDynamoDBAttributeValue handles custom unmarshaling of an ARN
func (a *ARN) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	if av.S == nil {
		return nil
	}

	err := a.parseString(*av.S)
	if err != nil {
		return err
	}
	return nil
}

// MarshalDynamoDBAttributeValue marshaling for DynamoDB
func (a *ARN) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	arnString := a.String()
	av.S = &arnString
	return nil
}

func (a *ARN) parseString(arnString string) error {
	arn, err := arn.Parse(arnString)
	if err != nil {
		return errors.NewInternalServer("unexpected error parsing arn", err)
	}
	a.Partition = arn.Partition
	a.Service = arn.Service
	a.Region = arn.Region
	a.AccountID = arn.AccountID
	a.Resource = arn.Resource
	return nil
}

// IAMResourceName returns the value from beyond the last /
// it will return nil if its not an IAM based arn
func (a *ARN) IAMResourceName() *string {

	if a.Service != "iam" {
		return nil
	}

	resourceName := strings.Split(a.Resource, "/")

	return &resourceName[len(resourceName)-1]
}

// NewFromArn creates a new ARN instance
func NewFromArn(arnString string) (*ARN, error) {
	new := &ARN{}
	err := new.parseString(arnString)
	if err != nil {
		return nil, errors.NewInternalServer("unexpexted error parsing arn", err)
	}
	return new, nil
}

// New creates a new ARN instance
func New(partition string, service string, region string, accountID string, resource string) *ARN {
	return &ARN{
		arn.ARN{
			Partition: partition,
			Service:   service,
			Region:    region,
			AccountID: accountID,
			Resource:  resource,
		},
	}
}
