package data

import (
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type stringer interface {
	String() string
}

func getFiltersFromStruct(input interface{}, keyName *string) (*expression.KeyConditionBuilder, *expression.ConditionBuilder) {
	var cb *expression.ConditionBuilder
	var kb *expression.KeyConditionBuilder
	var dValue interface{}
	v := reflect.ValueOf(input).Elem()
	for i := 0; i < v.NumField(); i++ {
		dField := strings.Split(v.Type().Field(i).Tag.Get("dynamodbav"), ",")[0]
		if dField != "-" {
			value := v.Field(i).Interface()
			if !reflect.ValueOf(value).IsNil() {
				t := v.Field(i)
				switch t.Kind() {
				case reflect.Ptr:
					if u, ok := t.Interface().(stringer); ok {
						dValue = u.String()
					} else {
						dValue = reflect.Indirect(v.Field(i)).Interface()
					}
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
	}
	return kb, cb
}

func putItem(input *dynamodb.PutItemInput, dataInterface dynamodbiface.DynamoDBAPI) error {
	_, err := dataInterface.PutItem(input)
	return err
}

func query(input *dynamodb.QueryInput, dataInterface dynamodbiface.DynamoDBAPI) (*dynamodb.QueryOutput, error) {
	output, err := dataInterface.Query(input)
	return output, err
}

func getItem(input *dynamodb.GetItemInput, dataInterface dynamodbiface.DynamoDBAPI) (*dynamodb.GetItemOutput, error) {
	output, err := dataInterface.GetItem(input)
	return output, err
}

func deleteItem(input *dynamodb.DeleteItemInput, dataInterface dynamodbiface.DynamoDBAPI) (*dynamodb.DeleteItemOutput, error) {
	output, err := dataInterface.DeleteItem(input)
	return output, err
}
