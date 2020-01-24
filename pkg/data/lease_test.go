package data

import (
	gErrors "errors"
	"fmt"
	"strconv"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetLeaseByAccountIDAndPrincipalID(t *testing.T) {
	tests := []struct {
		name          string
		accountID     string
		principalID   string
		dynamoErr     error
		dynamoOutput  *dynamodb.GetItemOutput
		expectedErr   error
		expectedLease *lease.Lease
	}{
		{
			name:        "should return a lease object",
			accountID:   "123456789012",
			principalID: "User1",
			expectedLease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"AccountId": {
						S: aws.String("123456789012"),
					},
					"PrincipalId": {
						S: aws.String("User1"),
					},
					"LeaseStatus": {
						S: aws.String("Active"),
					},
					"LastModifiedOn": {
						N: aws.String(strconv.Itoa(1573592058)),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:          "should return nil object when not found",
			accountID:     "123456789012",
			principalID:   "User1",
			expectedLease: nil,
			dynamoErr:     nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewNotFound("lease", "123456789012"),
		},
		{
			name:          "should return nil when dynamodb err",
			accountID:     "123456789012",
			principalID:   "User1",
			expectedLease: nil,
			dynamoErr:     gErrors.New("failure"),
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewInternalServer("get lease failed for account \"123456789012\" and principal \"User1\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("GetItem", mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
				return (*input.TableName == "Leases" &&
					*input.Key["AccountId"].S == tt.accountID &&
					*input.Key["PrincipalId"].S == tt.principalID)
			})).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			leaseData := &Lease{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
			}

			lease, err := leaseData.GetByAccountIDAndPrincipalID(tt.accountID, tt.principalID)
			assert.Equal(t, tt.expectedLease, lease)
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}

func TestLeaseDelete(t *testing.T) {

	tests := []struct {
		name         string
		lease        *lease.Lease
		dynamoErr    error
		dynamoOutput *dynamodb.DeleteItemOutput
		expectedErr  error
	}{
		{
			name: "should delete a lease successfully",
			lease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.DeleteItemOutput{
				Attributes: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: nil,
		},
		{
			name: "should delete a lease return error",
			lease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			dynamoErr: gErrors.New("failure"),
			dynamoOutput: &dynamodb.DeleteItemOutput{
				Attributes: map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewInternalServer("delete lease failed for account \"123456789012\" and principal \"User1\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("DeleteItem", mock.MatchedBy(func(input *dynamodb.DeleteItemInput) bool {
				return (*input.TableName == "Leases" &&
					*input.Key["AccountId"].S == *tt.lease.AccountID &&
					*input.Key["PrincipalId"].S == *tt.lease.PrincipalID)
			})).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			leaseData := &Lease{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
			}

			err := leaseData.Delete(tt.lease)
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}

func TestLeaseUpdate(t *testing.T) {
	tests := []struct {
		name              string
		lease             *lease.Lease
		dynamoErr         error
		expectedErr       error
		oldLastModifiedOn *int64
	}{
		{
			name: "update",
			lease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         nil,
			expectedErr:       nil,
		},
		{
			name: "create",
			lease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			dynamoErr:   nil,
			expectedErr: nil,
		},
		{
			name: "conditional failure",
			lease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         awserr.New("ConditionalCheckFailedException", "Message", fmt.Errorf("Bad")),
			expectedErr: errors.NewConflict(
				"lease",
				"123456789012",
				fmt.Errorf("unable to update lease: leases has been modified since request was made")),
		},
		{
			name: "other dynamo error",
			lease: &lease.Lease{
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User2"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			oldLastModifiedOn: ptrInt64(1573592057),
			dynamoErr:         gErrors.New("failure"),
			expectedErr:       errors.NewInternalServer("update failed for lease with AccountID \"123456789012\" and PrincipalID \"User2\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("PutItem", mock.Anything).Return(&dynamodb.PutItemOutput{}, tt.dynamoErr)

			mockDynamo.On("PutItem", mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
				if tt.oldLastModifiedOn == nil {
					return (*input.TableName == "Leases" &&
						*input.Item["AccountId"].S == *tt.lease.AccountID &&
						*input.Item["PrincipalId"].S == *tt.lease.PrincipalID &&
						input.Item["LeaseStatus"].S == tt.lease.Status.StringPtr() &&
						*input.Item["LastModifiedOn"].N == strconv.FormatInt(*tt.lease.LastModifiedOn, 10) &&
						*input.ConditionExpression == "attribute_not_exists (#0)")
				}
				return (*input.TableName == "Leases" &&
					*input.Item["AccountId"].S == *tt.lease.AccountID &&
					*input.Item["PrincipalId"].S == *tt.lease.PrincipalID &&
					input.Item["LeaseStatus"].S == tt.lease.Status.StringPtr() &&
					*input.Item["LastModifiedOn"].N == strconv.FormatInt(*tt.lease.LastModifiedOn, 10) &&
					*input.ExpressionAttributeValues[":0"].N == strconv.FormatInt(*tt.oldLastModifiedOn, 10))
			})).Return(
				&dynamodb.PutItemOutput{}, tt.dynamoErr,
			)
			leaseData := &Lease{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
			}

			err := leaseData.Write(tt.lease, tt.oldLastModifiedOn)
			assert.Truef(t, errors.Is(err, tt.expectedErr), "actual error %q doesn't match expected error %q", err, tt.expectedErr)
		})
	}

}

func TestGetLeaseByID(t *testing.T) {
	tests := []struct {
		name          string
		leaseID       string
		dynamoErr     error
		dynamoOutput  *dynamodb.QueryOutput
		expectedErr   error
		expectedLease *lease.Lease
	}{
		{
			name:    "should return a lease object",
			leaseID: "123",
			expectedLease: &lease.Lease{
				ID:             ptrString("123"),
				AccountID:      ptrString("123456789012"),
				PrincipalID:    ptrString("User1"),
				Status:         lease.StatusActive.StatusPtr(),
				LastModifiedOn: ptrInt64(1573592058),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"Id": {
							S: aws.String("123"),
						},
						"AccountId": {
							S: aws.String("123456789012"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
						"LeaseStatus": {
							S: aws.String("Active"),
						},
						"LastModifiedOn": {
							N: aws.String(strconv.Itoa(1573592058)),
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:          "should return nil when more than one found",
			leaseID:       "123",
			expectedLease: nil,
			dynamoErr:     nil,
			dynamoOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{
					{
						"Id": {
							S: aws.String("123"),
						},
						"AccountId": {
							S: aws.String("123456789012"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
						"LeaseStatus": {
							S: aws.String("Active"),
						},
						"LastModifiedOn": {
							N: aws.String(strconv.Itoa(1573592058)),
						},
					},
					{
						"Id": {
							S: aws.String("123"),
						},
						"AccountId": {
							S: aws.String("123456789012"),
						},
						"PrincipalId": {
							S: aws.String("User1"),
						},
						"LeaseStatus": {
							S: aws.String("Active"),
						},
						"LastModifiedOn": {
							N: aws.String(strconv.Itoa(1573592058)),
						},
					},
				},
			},
			expectedErr: errors.NewInternalServer("Found more than one Lease with id: \"123\"", gErrors.New("failure")),
		},
		{
			name:          "should return nil when dynamodb err",
			leaseID:       "123",
			expectedLease: nil,
			dynamoErr:     gErrors.New("failure"),
			dynamoOutput: &dynamodb.QueryOutput{
				Items: []map[string]*dynamodb.AttributeValue{},
			},
			expectedErr: errors.NewInternalServer("get lease failed for id \"123\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("Query", mock.MatchedBy(func(input *dynamodb.QueryInput) bool {
				return (*input.TableName == "Leases" &&
					*input.IndexName == "LeaseId" &&
					*input.KeyConditionExpression == "Id = :id" &&
					*input.ExpressionAttributeValues[":id"].S == tt.leaseID)
			})).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			leaseData := &Lease{
				DynamoDB:  &mockDynamo,
				TableName: "Leases",
			}

			lease, err := leaseData.Get(tt.leaseID)
			assert.Equal(t, tt.expectedLease, lease)
			assert.True(t, errors.Is(err, tt.expectedErr))
		})
	}

}
