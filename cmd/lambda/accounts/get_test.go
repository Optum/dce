package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

type mockData struct{}

func (d *mockData) GetAccountByID(accountID string, account *model.Account) error {
	if accountID == "error" {
		return errors.ErrInternalServer
	}

	readyAccount := model.Ready
	lastModifiedOn := int64(1561149393)

	account.ID = &accountID
	account.Status = &readyAccount
	account.LastModifiedOn = &lastModifiedOn
	return nil
}

func TestGetAccountByID(t *testing.T) {

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {

		accountID := "123456789"
		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: fmt.Sprintf("/accounts/%s", accountID)}

		mockDataImpl := &mockData{}
		cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1").Build()
		svcBldr := &config.ServiceBuilder{Config: cfgBldr}

		_, err := svcBldr.
			// AWS services...
			WithDynamoDB().
			WithSTS().
			WithS3().
			WithSNS().
			WithSQS().
			// DCE services...
			WithDAO().
			WithRoleManager().
			WithStorageService().
			WithDataService().
			WithAccountManager().
			Build()

		if err == nil {
			Services = svcBldr
		}
		actualResponse, err := Handler(context.TODO(), mockRequest)
		require.Nil(t, err)

		require.Equal(t, actualResponse.StatusCode, 200, "Returns a 200.")
	})

}
