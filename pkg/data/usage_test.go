package data

import (
	gErrors "errors"
	"strconv"
	"testing"

	awsmocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetUageByStartDateAndPrincipalID(t *testing.T) {
	tests := []struct {
		name         string
		startDate    int64
		principalID  string
		dynamoErr    error
		dynamoOutput *dynamodb.GetItemOutput
		expErr       error
		expUsage     *usage.Usage
	}{
		{
			name:        "should return a usage object",
			startDate:   1580924093,
			principalID: "User1",
			expUsage: &usage.Usage{
				StartDate:   ptrInt64(1580924093),
				PrincipalID: ptrString("User1"),
			},
			dynamoErr: nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"StartDate": {
						N: aws.String("1580924093"),
					},
					"PrincipalId": {
						S: aws.String("User1"),
					},
				},
			},
			expErr: nil,
		},
		{
			name:        "should return nil object when not found",
			startDate:   1580924093,
			principalID: "User1",
			dynamoErr:   nil,
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expUsage: nil,
			expErr:   errors.NewNotFound("usage", "1580924093-User1"),
		},
		{
			name:        "should return nil when dynamodb err",
			startDate:   1580924093,
			principalID: "User1",
			expUsage:    nil,
			dynamoErr:   gErrors.New("failure"),
			dynamoOutput: &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{},
			},
			expErr: errors.NewInternalServer("get usage failed for start date \"1580924093\" and principal \"User1\"", gErrors.New("failure")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDynamo := awsmocks.DynamoDBAPI{}

			mockDynamo.On("GetItem", mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
				return (*input.TableName == "Usage" &&
					*input.Key["StartDate"].N == strconv.FormatInt(tt.startDate, 10) &&
					*input.Key["PrincipalId"].S == tt.principalID)
			})).Return(
				tt.dynamoOutput, tt.dynamoErr,
			)
			usgData := &Usage{
				DynamoDB:  &mockDynamo,
				TableName: "Usage",
			}

			usg, err := usgData.GetByStartDateAndPrincipalID(tt.startDate, tt.principalID)
			assert.Equal(t, tt.expUsage, usg)
			assert.True(t, errors.Is(err, tt.expErr))
		})
	}

}
