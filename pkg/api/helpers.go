package api

import (
	"log"
	"net/url"

	"github.com/Optum/dce/pkg/errors"
	"github.com/gorilla/schema"
)

// GetStructFromQuery converts r query into a struct
func GetStructFromQuery(i interface{}, v url.Values) error {
	var decoder = schema.NewDecoder()

	err := decoder.Decode(i, v)
	if err != nil {
		log.Printf("%+v", err)
		return &errors.ErrValidation{
			Message: "error converting query parameters to struct",
			Err:     err,
		}
	}
	return nil
}
