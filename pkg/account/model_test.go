package account_test

import (
	"encoding/json"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/stretchr/testify/assert"
)

func TestAccountUnmarshalJSON(t *testing.T) {

	tests := []struct {
		name       string
		input      string
		expAccount *account.Account
	}{
		{
			name:  "should be able to unmarshal empty information",
			input: "{\"id\":\"123456789012\"}",
			expAccount: &account.Account{
				ID:                 ptrString("123456789012"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
			},
		},
		{
			name:  "should be able to unmarshal with principal role name",
			input: "{\"id\":\"123456789012\", \"principalRoleArn\": \"arn:aws:iam::123456789012:role/DCEPrincipal\"}",
			expAccount: &account.Account{
				ID:                 ptrString("123456789012"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acct := &account.Account{}
			err := json.Unmarshal([]byte(tt.input), acct)

			assert.Equal(t, nil, err)
			assert.Equal(t, tt.expAccount, acct)
		})
	}
}

func TestAccountMarshalJSON(t *testing.T) {

	tests := []struct {
		name       string
		input      *account.Account
		expAccount string
	}{
		{
			name:       "should be able to unmarshal empty information",
			expAccount: "{\"id\":\"123456789012\"}",
			input: &account.Account{
				ID:                 ptrString("123456789012"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
			},
		},
		{
			name:       "should be able to unmarshal with principal role name",
			expAccount: "{\"id\":\"123456789012\",\"principalRoleArn\":\"arn:aws:iam::123456789012:role/DCEPrincipal\"}",
			input: &account.Account{
				ID:                 ptrString("123456789012"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acct, err := json.Marshal(tt.input)

			assert.Equal(t, nil, err)
			assert.Equal(t, tt.expAccount, string(acct))
		})
	}
}

func TestAccountUnmarshalDynamoDB(t *testing.T) {

	tests := []struct {
		name       string
		input      *dynamodb.AttributeValue
		expAccount *account.Account
	}{
		{
			name: "should be able to unmarshal empty information",
			input: &dynamodb.AttributeValue{
				M: map[string]*dynamodb.AttributeValue{
					"ID": {
						S: aws.String("123456789012"),
					},
				},
			},
			expAccount: &account.Account{
				ID:                 ptrString("123456789012"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
			},
		},
		{
			name: "should be able to unmarshal with principal role name",
			input: &dynamodb.AttributeValue{
				M: map[string]*dynamodb.AttributeValue{
					"ID": {
						S: aws.String("123456789012"),
					},
					"PrincipalRoleArn": {
						S: aws.String("arn:aws:iam::123456789012:role/DCEPrincipal"),
					},
				},
			},
			expAccount: &account.Account{
				ID:                 ptrString("123456789012"),
				PrincipalPolicyArn: arn.New("aws", "iam", "", "123456789012", "policy/DCEPrincipalDefaultPolicy"),
				PrincipalRoleArn:   arn.New("aws", "iam", "", "123456789012", "role/DCEPrincipal"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acct := &account.Account{}
			err := dynamodbattribute.Unmarshal(tt.input, acct)

			assert.Equal(t, nil, err)
			assert.Equal(t, tt.expAccount, acct)
		})
	}
}
