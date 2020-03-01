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

type unixer interface {
	Unix() int64
}

type sortKey struct {
	keyName    string
	typeSearch string
}

func getFiltersFromStruct(input interface{}, partitionKey *string, sortKey *sortKey) (*expression.KeyConditionBuilder, *expression.ConditionBuilder) {
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
					if u, ok := t.Interface().(unixer); ok {
						dValue = u.Unix()
					} else if u, ok := t.Interface().(stringer); ok {
						dValue = u.String()
					} else {
						dValue = reflect.Indirect(v.Field(i)).Interface()
					}
					if partitionKey != nil {
						if dField == *partitionKey {
							kb = kbAdd(kb, expression.Key(dField).Equal(expression.Value(dValue)))
							continue
						}
					}
					if sortKey != nil {
						if dField == sortKey.keyName {
							switch typeSearch := sortKey.typeSearch; typeSearch {
							case "Equal":
								kb = kbAdd(kb, expression.Key(dField).Equal(expression.Value(dValue)))
							case "BeginsWith":
								if s, ok := dValue.(string); ok {
									kb = kbAdd(kb, expression.Key(dField).BeginsWith(s))
								}
							}
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

func kbAdd(kb *expression.KeyConditionBuilder, condition expression.KeyConditionBuilder) *expression.KeyConditionBuilder {
	if kb == nil {
		return &condition
	}
	new := kb.And(condition)
	return &new
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
