package data

import (
	"testing"
	"time"

	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func ptrFloat64(f float64) *float64 {
	ptrF := f
	return &ptrF
}

func TestWriteLeaseUsage(t *testing.T) {

	theTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	type expPutItem struct {
		input  *dynamodb.PutItemInput
		output *dynamodb.PutItemOutput
		outErr error
	}

	type expUpdateItem struct {
		input  *dynamodb.UpdateItemInput
		output *dynamodb.UpdateItemOutput
		outErr error
	}

	tests := []struct {
		name              string
		req               usage.Lease
		putLeaseUsage     expPutItem
		addLeaseUsage     expUpdateItem
		addPrincipalUsage expUpdateItem
		expErr            error
	}{
		{
			name: "should upsert usage information",
			req: usage.Lease{
				Date:         &theTime,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(1.0),
				CostCurrency: ptrString("USD"),
			},
			putLeaseUsage: expPutItem{
				input: &dynamodb.PutItemInput{
					TableName: aws.String("table"),
					Item: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"LeaseId": {
							S: aws.String("id-1"),
						},
						"Date": {
							N: aws.String("1577836800"),
						},
						"CostAmount": {
							N: aws.String("1"),
						},
						"CostCurrency": {
							S: aws.String("USD"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-1577836800"),
						},
						"TimeToLive": {
							N: aws.String("1577836800"),
						},
					},
					ReturnValues: aws.String("ALL_OLD"),
				},
				output: &dynamodb.PutItemOutput{
					Attributes: map[string]*dynamodb.AttributeValue{},
				},
			},
			addLeaseUsage: expUpdateItem{
				input: &dynamodb.UpdateItemInput{
					ExpressionAttributeNames: map[string]*string{
						"#0": aws.String("CostAmount"),
						"#1": aws.String("CostCurrency"),
						"#2": aws.String("Date"),
						"#3": aws.String("TimeToLive"),
					},
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":1": {
							S: aws.String("USD"),
						},
						":2": {
							N: aws.String("1577836800"),
						},
						":3": {
							N: aws.String("1577836800"),
						},
						":0": {
							N: aws.String("1"),
						},
					},
					Key: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-Summary"),
						},
					},
					ReturnValues:     aws.String("ALL_NEW"),
					TableName:        aws.String("table"),
					UpdateExpression: aws.String("ADD #0 :0\nSET #1 = :1, #2 = :2, #3 = :3\n"),
				},
				output: &dynamodb.UpdateItemOutput{},
				outErr: nil,
			},
			addPrincipalUsage: expUpdateItem{
				input: &dynamodb.UpdateItemInput{
					ExpressionAttributeNames: map[string]*string{
						"#0": aws.String("CostAmount"),
						"#1": aws.String("CostCurrency"),
						"#2": aws.String("Date"),
						"#3": aws.String("TimeToLive"),
					},
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":1": {
							S: aws.String("USD"),
						},
						":2": {
							N: aws.String("1577664000"),
						},
						":3": {
							N: aws.String("1577836800"),
						},
						":0": {
							N: aws.String("1"),
						},
					},
					Key: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"SK": {
							S: aws.String("Usage-Principal-1577664000"),
						},
					},
					ReturnValues:     aws.String("ALL_NEW"),
					TableName:        aws.String("table"),
					UpdateExpression: aws.String("ADD #0 :0\nSET #1 = :1, #2 = :2, #3 = :3\n"),
				},
				output: &dynamodb.UpdateItemOutput{},
				outErr: nil,
			},
			expErr: nil,
		},
		{
			name: "should add no usage when no change",
			req: usage.Lease{
				Date:         &theTime,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(5.0),
				CostCurrency: ptrString("USD"),
			},
			putLeaseUsage: expPutItem{
				input: &dynamodb.PutItemInput{
					TableName: aws.String("table"),
					Item: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"LeaseId": {
							S: aws.String("id-1"),
						},
						"Date": {
							N: aws.String("1577836800"),
						},
						"CostAmount": {
							N: aws.String("5"),
						},
						"CostCurrency": {
							S: aws.String("USD"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-1577836800"),
						},
						"TimeToLive": {
							N: aws.String("1577836800"),
						},
					},
					ReturnValues: aws.String("ALL_OLD"),
				},
				output: &dynamodb.PutItemOutput{
					Attributes: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"LeaseId": {
							S: aws.String("id-1"),
						},
						"Date": {
							N: aws.String("1577836800"),
						},
						"CostAmount": {
							N: aws.String("5.0"),
						},
						"CostCurrency": {
							S: aws.String("USD"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-1577836800"),
						},
						"TimeToLive": {
							N: aws.String("1577836800"),
						},
					},
				},
			},
			addLeaseUsage: expUpdateItem{
				input:  &dynamodb.UpdateItemInput{},
				output: &dynamodb.UpdateItemOutput{},
				outErr: nil,
			},
			addPrincipalUsage: expUpdateItem{
				input:  &dynamodb.UpdateItemInput{},
				output: &dynamodb.UpdateItemOutput{},
				outErr: nil,
			},
			expErr: nil,
		},
		{
			name: "should add usage information",
			req: usage.Lease{
				Date:         &theTime,
				PrincipalID:  ptrString("test"),
				LeaseID:      ptrString("id-1"),
				CostAmount:   ptrFloat64(5.0),
				CostCurrency: ptrString("USD"),
			},
			putLeaseUsage: expPutItem{
				input: &dynamodb.PutItemInput{
					TableName: aws.String("table"),
					Item: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"LeaseId": {
							S: aws.String("id-1"),
						},
						"Date": {
							N: aws.String("1577836800"),
						},
						"CostAmount": {
							N: aws.String("5"),
						},
						"CostCurrency": {
							S: aws.String("USD"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-1577836800"),
						},
						"TimeToLive": {
							N: aws.String("1577836800"),
						},
					},
					ReturnValues: aws.String("ALL_OLD"),
				},
				output: &dynamodb.PutItemOutput{
					Attributes: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"LeaseId": {
							S: aws.String("id-1"),
						},
						"Date": {
							N: aws.String("1577836800"),
						},
						"CostAmount": {
							N: aws.String("1.0"),
						},
						"CostCurrency": {
							S: aws.String("USD"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-1577836800"),
						},
						"TimeToLive": {
							N: aws.String("1577836800"),
						},
					},
				},
			},
			addLeaseUsage: expUpdateItem{
				input: &dynamodb.UpdateItemInput{
					ExpressionAttributeNames: map[string]*string{
						"#0": aws.String("CostAmount"),
						"#1": aws.String("CostCurrency"),
						"#2": aws.String("Date"),
						"#3": aws.String("TimeToLive"),
					},
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":1": {
							S: aws.String("USD"),
						},
						":2": {
							N: aws.String("1577836800"),
						},
						":3": {
							N: aws.String("1577836800"),
						},
						":0": {
							N: aws.String("4"),
						},
					},
					Key: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"SK": {
							S: aws.String("Usage-Lease-id-1-Summary"),
						},
					},
					ReturnValues:     aws.String("ALL_NEW"),
					TableName:        aws.String("table"),
					UpdateExpression: aws.String("ADD #0 :0\nSET #1 = :1, #2 = :2, #3 = :3\n"),
				},
				output: &dynamodb.UpdateItemOutput{},
				outErr: nil,
			},
			addPrincipalUsage: expUpdateItem{
				input: &dynamodb.UpdateItemInput{
					ExpressionAttributeNames: map[string]*string{
						"#0": aws.String("CostAmount"),
						"#1": aws.String("CostCurrency"),
						"#2": aws.String("Date"),
						"#3": aws.String("TimeToLive"),
					},
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":1": {
							S: aws.String("USD"),
						},
						":2": {
							N: aws.String("1577664000"),
						},
						":3": {
							N: aws.String("1577836800"),
						},
						":0": {
							N: aws.String("4"),
						},
					},
					Key: map[string]*dynamodb.AttributeValue{
						"PrincipalId": {
							S: aws.String("test"),
						},
						"SK": {
							S: aws.String("Usage-Principal-1577664000"),
						},
					},
					ReturnValues:     aws.String("ALL_NEW"),
					TableName:        aws.String("table"),
					UpdateExpression: aws.String("ADD #0 :0\nSET #1 = :1, #2 = :2, #3 = :3\n"),
				},
				output: &dynamodb.UpdateItemOutput{},
				outErr: nil,
			},
			expErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksDynamo := &awsMocks.DynamoDBAPI{}

			mocksDynamo.On("PutItem", tt.putLeaseUsage.input).Return(tt.putLeaseUsage.output, tt.putLeaseUsage.outErr)
			mocksDynamo.On("UpdateItem", tt.addLeaseUsage.input).Return(tt.addLeaseUsage.output, tt.addLeaseUsage.outErr)
			mocksDynamo.On("UpdateItem", tt.addPrincipalUsage.input).Return(tt.addPrincipalUsage.output, tt.addPrincipalUsage.outErr)

			dataSvc := UsageLease{
				DynamoDB:  mocksDynamo,
				TableName: "table",
			}

			err := dataSvc.Write(&tt.req)
			assert.True(t, errors.Is(err, tt.expErr), "actual error %q doesn't match expected error %q", err, tt.expErr)
		})
	}
}
