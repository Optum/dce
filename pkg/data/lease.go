package data

import (
	"fmt"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// Write the Lease record in DynamoDB
// This is an upsert operation in which the record will either
// be inserted or updated
// prevLastModifiedOn parameter is the original lastModifiedOn
func (a *Lease) Write(lease *lease.Lease, prevLastModifiedOn *int64) error {

	var expr expression.Expression
	var err error
	returnValue := "NONE"
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

	putMap, _ := dynamodbattribute.Marshal(lease)
	input := &dynamodb.PutItemInput{
		TableName:                 aws.String(a.TableName),
		Item:                      putMap.M,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              aws.String(returnValue),
	}
	err = putItem(input, a.DynamoDB)
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		if awsErr.Code() == "ConditionalCheckFailedException" {
			return errors.NewConflict(
				"lease",
				*lease.AccountID,
				fmt.Errorf("unable to update lease: leases has been modified since request was made"))
		}
	}
	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("update failed for lease with AccountID %q and PrincipalID %q", *lease.AccountID, *lease.PrincipalID),
			err,
		)
	}

	return nil

}

// Delete the Lease record in DynamoDB
func (a *Lease) Delete(lease *lease.Lease) error {

	input := &dynamodb.DeleteItemInput{
		// Query in Lease Table
		TableName: aws.String(a.TableName),
		// Return the updated record
		ReturnValues: aws.String("ALL_NEW"),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountId": {
				S: lease.AccountID,
			},
			"PrincipalId": {
				S: lease.PrincipalID,
			},
		},
	}
	_, err := deleteItem(input, a.DynamoDB)

	if err != nil {
		return errors.NewInternalServer(
			fmt.Sprintf("delete lease failed for account %q and principal %q", *lease.AccountID, *lease.PrincipalID),
			err,
		)
	}

	return nil
}

// GetByAccountIDAndPrincipalID gets the Lease record by AccountID and PrincipalID
func (a *Lease) GetByAccountIDAndPrincipalID(accountID string, principalID string) (*lease.Lease, error) {

	input := &dynamodb.GetItemInput{
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
	}

	res, err := getItem(input, a.DynamoDB)

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get lease failed for account %q and principal %q", accountID, principalID),
			err,
		)
	}

	if len(res.Item) == 0 {
		return nil, errors.NewNotFound("lease", accountID)
	}

	lease := lease.Lease{}
	err = dynamodbattribute.UnmarshalMap(res.Item, &lease)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling lease with account %q and princiapl %q", accountID, principalID),
			err,
		)
	}
	return &lease, nil
}

// GetByID gets the Lease record by ID
func (a *Lease) GetByID(leaseID string) (*lease.Lease, error) {

	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(leaseID),
			},
		},
		KeyConditionExpression: aws.String("Id = :id"),
		TableName:              aws.String(a.TableName),
		IndexName:              aws.String("LeaseId"),
		ConsistentRead:         aws.Bool(a.ConsistentRead),
	}
	res, err := query(input, a.DynamoDB)

	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("get lease failed for id %q", leaseID),
			err,
		)
	}

	if len(res.Items) == 0 {
		return nil, errors.NewNotFound("lease", leaseID)
	}

	if len(res.Items) > 1 {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("Found more than one Lease with id: %q", leaseID),
			err,
		)
	}

	lease := lease.Lease{}
	err = dynamodbattribute.UnmarshalMap(res.Items[0], &lease)
	if err != nil {
		return nil, errors.NewInternalServer(
			fmt.Sprintf("failure unmarshaling lease with id %q", leaseID),
			err,
		)
	}
	return &lease, nil
}
