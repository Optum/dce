package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// Lease - Data Layer Struct
type Lease struct {
	DynamoDB       dynamodbiface.DynamoDBAPI
	TableName      string `env:"LEASE_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
}

// WriteLease the Lease record in DynamoDB
// This is an upsert operation in which the record will either
// be inserted or updated
// prevLastModifiedOn parameter is the original lastModifiedOn
func (a *Lease) WriteLease(lease *model.Lease, prevLastModifiedOn *int64) error {

	var expr expression.Expression
	var err error
	// lastModifiedOn is nil on a create
	if prevLastModifiedOn != nil {
		modExpr := expression.Name("LastModifiedOn").Equal(expression.Value(prevLastModifiedOn))
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		if err != nil {
			return errors.NewInternalServer("error building query", err)
		}
	} else {
		modExpr := expression.Name("LastModifiedOn").AttributeNotExists()
		expr, err = expression.NewBuilder().WithCondition(modExpr).Build()
		if err != nil {
			return errors.NewInternalServer("error building query", err)
		}
	}

	return putItem(a, lease, "lease", &expr)

}

// DeleteLease the Lease record in DynamoDB
func (a *Lease) DeleteLease(lease *model.Lease) error {

	_, err := a.DynamoDB.DeleteItem(
		&dynamodb.DeleteItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			// Return the updated record
			ReturnValues: aws.String("ALL_NEW"),
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: &lease.AccountID,
				},
				"PrincipalId": {
					S: &lease.PrincipalID,
				},
			},
		},
	)

	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("delete lease failed for account %q and principal %q", &lease.AccountID, &lease.PrincipalID),
			err,
		)
	}

	return nil
}

// GetLeaseByID the Lease record by ID
func (a *Lease) GetLeaseByAccountIDAndPrincipalID(accountID string, principalID string) (*model.Lease, error) {

	res, err := a.DynamoDB.GetItem(
		&dynamodb.GetItemInput{
			// Query in Lease Table
			TableName: aws.String(a.TableName),
			Key: map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: aws.String(accountID),
				},
				"PrincipalId": {
					S: aws.String(principalID),
				},
			},
			ConsistentRead: aws.Bool(a.ConsistentRead),
		},
	)

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get lease failed for account %q and principal %q", accountID, principalID),
			err,
		)
	}

	if len(res.Item) == 0 {
		return nil, errors.NewNotFound("lease", accountID)
	}

	lease := model.Lease{}
	err = dynamodbattribute.UnmarshalMap(res.Item, &lease)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling lease with account %q and princiapl %q", accountID, principalID),
			err,
		)
	}
	return &lease, nil
}
