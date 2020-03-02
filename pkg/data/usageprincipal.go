package data

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const usagePrincipalSkPrefix string = "Usage-Principal-"

type usagePrincipalData struct {
	usage.Principal
	SK         string `json:"-" dynamodbav:"SK" schema:"-"`
	TimeToLive *int64 `json:"timeToLive" dynamodbav:"TimeToLive,omitempty" schema:"timeToLive,omitempty"` // ttl attribute
}

// UsagePrincipal - Data Layer Struct
type UsagePrincipal struct {
	DynamoDB       dynamodbiface.DynamoDBAPI
	TableName      string `env:"PRINCIPAL_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
	Limit          int64  `env:"LIMIT" envDefault:"25"`
	BudgetPeriod   string `env:"PRINCIPAL_BUDGET_PERIOD" envDefault:"WEEKLY"`
	TimeToLive     int    `env:"USAGE_TTL" envDefault:"30"`
}

// Get usage Principal summary
func (a *UsagePrincipal) Get(id string, date time.Time) (*usage.Principal, error) {
	res, err := a.DynamoDB.GetItem(
		&dynamodb.GetItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			Key: map[string]*dynamodb.AttributeValue{
				"PrincipalId": {
					S: aws.String(id),
				},
				"SK": {
					S: aws.String(fmt.Sprintf("%s%s", usagePrincipalSkPrefix, strconv.FormatInt(date.UTC().Unix(), 10))),
				},
			},
			ConsistentRead: aws.Bool(a.ConsistentRead),
		},
	)

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get failed for usage %q", id),
			err,
		)
	}

	if len(res.Item) == 0 {
		return nil, errors.NewNotFound("usage", id)
	}

	usg := &usage.Principal{}
	err = dynamodbattribute.UnmarshalMap(res.Item, usg)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling usage %q", id),
			err,
		)
	}
	return usg, nil
}