package data

import (
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/service/dynamodb"
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

func putItem(input *dynamodb.PutItemInput, dataInterface *Account) error {
	_, err := dataInterface.DynamoDB.PutItem(input)
	return err
}
