package common

import (
	"net/http"
)

// HTTPClienter interface requires a method to execute an http request and
// return the http response
type HTTPClienter interface {
	Do(*http.Request) (*http.Response, error)
}

// HTTPClient implements the HTTPClienter interface with net/http
type HTTPClient struct {
	Client http.Client
}

// Do executes and returns the http request's response
func (client *HTTPClient) Do(request *http.Request) (*http.Response, error) {
	return client.Client.Do(request)
}
