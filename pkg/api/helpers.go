package api

import (
	"fmt"
	"net/url"

	"github.com/Optum/dce/pkg/errors"
	"github.com/gorilla/schema"
)

// BuildNextURL merges the next parameters of pagination into the request parameters and returns an API URL.
func BuildNextURL(u url.URL, i interface{}) (url.URL, error) {
	req := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	values := url.Values{}
	err := schema.NewEncoder().Encode(i, values)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return url.URL{}, errors.NewInternalServer("unable to encode query", err)
	}

	req.RawQuery = values.Encode()
	return req, nil
}
