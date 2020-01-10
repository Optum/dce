package data

import (
	"github.com/Optum/dce/pkg/model"
	"reflect"
	"strings"

	gErrors "errors"
	"fmt"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func getFiltersFromStruct(i interface{}, keyName *string) (*expression.KeyConditionBuilder, *expression.ConditionBuilder) {
	var cb *expression.ConditionBuilder
	var kb *expression.KeyConditionBuilder
	v := reflect.ValueOf(i).Elem()
	for i := 0; i < v.NumField(); i++ {
		dField := strings.Split(v.Type().Field(i).Tag.Get("dynamodbav"), ",")[0]
		value := v.Field(i).Interface()
		if !reflect.ValueOf(value).IsNil() {
			switch v.Field(i).Kind() {
			case reflect.Ptr:
				dValue := reflect.Indirect(v.Field(i)).Interface()
				if keyName != nil {
					if dField == *keyName {
						newFilter := expression.Key(dField).Equal(expression.Value(dValue))
						kb = &newFilter
						continue
					}
				}
				if cb == nil {
					newFilter := expression.Name(dField).Equal(expression.Value(dValue))
					cb = &newFilter
				} else {
					*cb = cb.And(expression.Name(dField).Equal(expression.Value(dValue)))
				}

			}
		}
	}
	return kb, cb
}

func putItem(input interface{}, data interface{}, tableName string, expr *expression.Expression) error {

	returnValue := "NONE"
	i, _ := input.(*Account)
	account, _ := data.(model.Account)

	putMap, _ := dynamodbattribute.Marshal(account)
	err := Invoke(i.DynamoDB, "PutItem",
		&dynamodb.PutItemInput{
			// Query in input Table
			TableName: aws.String(tableName),
			// Find record for the requested input
			Item: putMap.M,
			// Condition Expression
			ConditionExpression:       expr.Condition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			// Return the updated record
			ReturnValues: aws.String(returnValue),
		},
	)
	var awsErr awserr.Error
	if gErrors.As(err, &awsErr) {
		if awsErr.Code() == "ConditionalCheckFailedException" {
			return errors.NewConflict(
				tableName,
				"input",
				fmt.Errorf("unable to update %s: Table has been modified since request was made", tableName))
		}
	}

	return nil
}

func Invoke(any interface{}, name string, args... interface{}) error {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	result := reflect.ValueOf(any).MethodByName(name).Call(inputs)
	err := result[0].Interface()

	if err != nil {
		return nil
	}

	return gErrors.New("error")
}