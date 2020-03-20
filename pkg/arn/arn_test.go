package arn

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/stretchr/testify/assert"
)

func TestNewFromString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expArn *ARN
		expErr error
	}{
		{
			name:  "when valid arn provided. An appropriate ARN object is returned.",
			input: "arn:aws:iam::123456789012:role/test",
			expArn: &ARN{
				arn.ARN{
					Partition: "aws",
					Service:   "iam",
					AccountID: "123456789012",
					Resource:  "role/test",
				},
			},
			expErr: nil,
		},
		{
			name:   "when an invalid arn provided. An error is returned.",
			input:  "arn:aws:iam::role/test",
			expArn: nil,
			expErr: errors.NewInternalServer("unexpexted error parsing arn", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			newArn, err := NewFromArn(tt.input)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
			assert.Equal(t, tt.expArn, newArn)
		})
	}
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name      string
		a         *ARN
		expString string
		expErr    error
	}{
		{
			name:      "when valid arn provided. An appropriate ARN object is returned.",
			a:         New("aws", "iam", "", "123456789012", "role/test"),
			expString: strconv.Quote("arn:aws:iam::123456789012:role/test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			b, err := json.Marshal(tt.a)
			assert.Equal(t, tt.expString, string(b))
			assert.Equal(t, tt.expErr, err)
		})
	}

}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expArn *ARN
		expErr error
	}{
		{
			name:   "when valid arn provided. An appropriate ARN object is returned.",
			expArn: New("aws", "iam", "", "123456789012", "role/test"),
			input:  []byte(strconv.Quote("arn:aws:iam::123456789012:role/test")),
		},
		{
			name:   "when an invalid arn is provided. An appropriate error is returned.",
			expArn: New("", "", "", "", ""),
			input:  []byte(strconv.Quote("arn:aws:iam::role/test")),
			expErr: errors.NewInternalServer("unexpected error parsing arn", fmt.Errorf("arn: not enough sections")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ARN{}
			err := json.Unmarshal(tt.input, &a)
			assert.Equal(t, tt.expArn, a)
			assert.True(t, errors.Is(err, tt.expErr))
		})
	}

}

func TestDynamoDBMarhal(t *testing.T) {
	tests := []struct {
		name      string
		a         *ARN
		expAttMap *dynamodb.AttributeValue
		expErr    error
	}{
		{
			name: "when valid arn provided. An appropriate ARN object is returned.",
			a:    New("aws", "iam", "", "123456789012", "role/test"),
			expAttMap: &dynamodb.AttributeValue{
				S: aws.String("arn:aws:iam::123456789012:role/test"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attMap, err := dynamodbattribute.Marshal(tt.a)
			assert.Equal(t, tt.expAttMap, attMap)
			assert.True(t, errors.Is(err, tt.expErr))
		})
	}

}

func TestDynamoDBUnmarhal(t *testing.T) {
	tests := []struct {
		name   string
		expArn *ARN
		input  *dynamodb.AttributeValue
		expErr error
	}{
		{
			name:   "when valid arn provided. An appropriate ARN object is returned.",
			expArn: New("aws", "iam", "", "123456789012", "role/test"),
			input: &dynamodb.AttributeValue{
				S: aws.String("arn:aws:iam::123456789012:role/test"),
			},
		},
		{
			name:   "when invalid arn provided. An appropriate error returned.",
			expArn: New("", "", "", "", ""),
			input: &dynamodb.AttributeValue{
				S: aws.String("arn:aws:iam::role/test"),
			},
			expErr: errors.NewInternalServer("unexpected error parsing arn", fmt.Errorf("arn: not enough sections")),
		},
		{
			name:   "when an empty arn is provided. An empty ARN response returned.",
			expArn: New("", "", "", "", ""),
			input: &dynamodb.AttributeValue{
				S: nil,
			},
			expErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ARN{}
			err := dynamodbattribute.Unmarshal(tt.input, a)
			assert.Equal(t, tt.expArn, a)
			assert.True(t, errors.Is(err, tt.expErr))
		})
	}

}

func TestIAMResourceName(t *testing.T) {
	tests := []struct {
		name string
		arn  *ARN
		exp  *string
	}{
		{
			name: "when a valid iam arn is provied. An appropriate resource name is provided.",
			arn:  New("aws", "iam", "", "123456789012", "role/test"),
			exp:  aws.String("test"),
		},
		{
			name: "when a valid iam arn is provied with additional paths. An appropriate resource name is provided.",
			arn:  New("aws", "iam", "", "123456789012", "role/path1/path2/path3/test"),
			exp:  aws.String("test"),
		},
		{
			name: "when a non iam resource is provided. A nil is returned.",
			arn:  New("aws", "sns", "us-east-1", "123456789012", "topic"),
			exp:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.exp, tt.arn.IAMResourceName())
		})
	}

}
