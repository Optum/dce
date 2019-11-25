package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

func TestGetAuth(t *testing.T) {

	t.Run("When invoke /auth and there are no errors then repond with html", func(t *testing.T) {
		// Arrange
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/auth"}

		// Act
		actualResponse, err := Handler(context.TODO(), mockRequest)
		require.Nil(t, err)

		// Assert
		require.Contains(t, actualResponse.Body, "<html>", "Returns html page")
		require.Contains(t, actualResponse.Body, "identityPoolID", "Template variables are rendered to default values")
		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

	t.Run("When invoke /auth/public/* and there are no errors then repond with static assets", func(t *testing.T) {
		// Arrange
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/auth/public/main.js"}

		// Act
		actualResponse, err := Handler(context.TODO(), mockRequest)
		require.Nil(t, err)

		// Assert
		wd, _ := os.Getwd()
		jsPath := filepath.Join(wd, "public", "main.js")
		jsFile := readFile(jsPath)
		require.Equal(t, 200, actualResponse.StatusCode, "Returns a 200.")
		require.Equal(t, actualResponse.Body, jsFile, "Returns js file")
	})
}

func readFile(path string) string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	return string(b)
}
