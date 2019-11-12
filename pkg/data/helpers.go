package data

import (
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func getFiltersFromStruct(i interface{}) *expression.ConditionBuilder {
	var filt *expression.ConditionBuilder
	v := reflect.ValueOf(i).Elem()
	for i := 0; i < v.NumField(); i++ {
		dField := strings.Split(v.Type().Field(i).Tag.Get("dynamodbav"), ",")[0]
		value := v.Field(i).Interface()
		if !reflect.ValueOf(value).IsNil() {
			switch v.Field(i).Kind() {
			case reflect.Ptr:
				dValue := reflect.Indirect(v.Field(i)).Interface()
				if filt == nil {
					newFilter := expression.Name(dField).Equal(expression.Value(dValue))
					filt = &newFilter
				} else {
					*filt = filt.And(expression.Name(dField).Equal(expression.Value(dValue)))
				}
			}
		}
	}
	return filt
}
