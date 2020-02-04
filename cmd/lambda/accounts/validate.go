package main

import (
	"errors"
	"reflect"
)

func isNil(value interface{}) error {
	if !reflect.ValueOf(value).IsNil() {
		return errors.New("must be empty")
	}
	return nil
}
