package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/db"
	util "github.com/Optum/dce/tests/testutils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateAccountByID(t *testing.T) {

	t.Run("should update the account adminRole and metadata", func(t *testing.T) {
		stubAllServices()
		dbMock := dbStub()
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&dbMock)

		// Should update the account
		util.ReplaceMock(&dbMock.Mock,
			"UpdateAccount",
			db.Account{
				ID:           "123456789012",
				AdminRoleArn: "new:role:arn",
				Metadata:     map[string]interface{}{"foo": "bar"},
			},
			[]string{"AdminRoleArn", "Metadata"},
		).Return(&db.Account{
			ID:           "123456789012",
			AdminRoleArn: "new:role:arn",
			Metadata:     map[string]interface{}{"foo": "bar"},
			// other fields, we don't really care about
			// but which will be included in the API response
			AccountStatus:       db.Ready,
			PrincipalRoleArn:    "prolearn",
			PrincipalPolicyHash: "phash",
			CreatedOn:           100,
			LastModifiedOn:      200,
		}, nil)

		// Call the controller
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123456789012", map[string]interface{}{
				"adminRoleArn": "new:role:arn",
				"metadata":     map[string]interface{}{"foo": "bar"},
			}),
		)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)

		// Check the response body
		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"id":           "123456789012",
			"adminRoleArn": "new:role:arn",
			"metadata": map[string]interface{}{
				"foo": "bar",
			},
			"accountStatus":       "Ready",
			"principalRoleArn":    "prolearn",
			"principalPolicyHash": "phash",
			"createdOn":           100.0,
			"lastModifiedOn":      200.0,
		}, resJSON)

		dbMock.AssertNumberOfCalls(t, "UpdateAccount", 1)
	})

	t.Run("should update just adminRoleArn", func(t *testing.T) {
		stubAllServices()
		dbMock := dbStub()
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&dbMock)

		// Should update the AdminRoleArn only
		util.ReplaceMock(&dbMock.Mock,
			"UpdateAccount",
			db.Account{
				ID:           "123456789012",
				AdminRoleArn: "new:role:arn",
			},
			[]string{"AdminRoleArn"},
		).Return(&db.Account{}, nil)

		// Call the controller
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123456789012", map[string]interface{}{
				"adminRoleArn": "new:role:arn",
			}),
		)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)

		// Check the dbmock was called
		dbMock.AssertNumberOfCalls(t, "UpdateAccount", 1)
	})

	t.Run("should update just metadata", func(t *testing.T) {
		stubAllServices()
		dbMock := dbStub()
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&dbMock)

		// Should update the metadata only
		util.ReplaceMock(&dbMock.Mock,
			"UpdateAccount",
			db.Account{
				ID:       "123456789012",
				Metadata: map[string]interface{}{"foo": "bar"},
			},
			[]string{"Metadata"},
		).Return(&db.Account{}, nil)

		// Call the controller
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123456789012", map[string]interface{}{
				"metadata": map[string]interface{}{"foo": "bar"},
			}),
		)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)

		// Check the dbmock was called
		dbMock.AssertNumberOfCalls(t, "UpdateAccount", 1)
	})

	t.Run("should allow you to pass in a full account object, without updating non-updatable fields", func(t *testing.T) {
		stubAllServices()
		dbMock := dbStub()
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&dbMock)

		// Should update the metadata only (not other account fields)
		util.ReplaceMock(&dbMock.Mock,
			"UpdateAccount",
			db.Account{
				ID:       "123456789012",
				Metadata: map[string]interface{}{"foo": "bar"},
			},
			[]string{"Metadata"},
		).Return(&db.Account{
			AccountStatus: db.NotReady,
		}, nil)

		// Call the controller
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123456789012", map[string]interface{}{
				// We'll update the adminRoleArn,
				// but pass in other fields too.
				// Controller should ignore these other fields entirely.
				// But this is useful for clients are keeping a client-side
				// representation of the data round.
				"metadata":      map[string]interface{}{"foo": "bar"},
				"accountStatus": "Ready",
				"createdOn":     100,
				// etc.
			}),
		)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)

		// Check the dbmock was called
		dbMock.AssertNumberOfCalls(t, "UpdateAccount", 1)
	})

	t.Run("should fail for invalid JSON", func(t *testing.T) {
		// Call the controller with invalid JSON
		res, err := Handler(context.TODO(),
			events.APIGatewayProxyRequest{
				HTTPMethod: "PUT",
				Path:       "/accounts/123456789012",
				Body:       "not json",
			},
		)
		require.Nil(t, err)

		require.Equal(t, 400, res.StatusCode)

		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "ClientError",
				"message": "invalid request parameters",
			},
		}, resJSON)
	})

	t.Run("should 404 if the account doesn't exist", func(t *testing.T) {
		// Mock the DB to return an NotFound error
		stubAllServices()
		dbMock := dbStub()
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&dbMock)

		// Should update the metadata only
		util.ReplaceMock(&dbMock.Mock,
			"UpdateAccount",
			mock.Anything,
			mock.Anything,
		).Return(nil, &db.NotFoundError{
			fmt.Sprintf(
				"Unable to update account 123456789012: account does not exist",
			),
		},
		)

		// Call the controller
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123456789012", map[string]interface{}{
				"metadata": map[string]interface{}{"foo": "bar"},
			}),
		)
		require.Nil(t, err)

		require.Equal(t, 404, res.StatusCode)

		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "NotFound",
				"message": "The requested resource could not be found.",
			},
		}, resJSON)
	})

	t.Run("should fail for invalid request field", func(t *testing.T) {
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123", map[string]interface{}{
				"foo": "bar",
			}),
		)
		require.Nil(t, err)

		require.Equal(t, 400, res.StatusCode)

		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "RequestValidationError",
				"message": "Unable to update account 123: no updatable fields provided",
			},
		}, resJSON)
	})

	t.Run("should fail if the admin role can't be assumed", func(t *testing.T) {
		stubAllServices()
		tokenSvc := &commonMocks.TokenService{}
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&tokenSvc)

		// Mock the TokenSvc, to fail on assume role
		util.ReplaceMock(&tokenSvc.Mock,
			"AssumeRole",
			&sts.AssumeRoleInput{
				RoleArn:         aws.String("new:admin:role"),
				RoleSessionName: aws.String("MasterAssumeRoleVerification"),
			},
		).Return(nil, errors.New("assume role failed"))

		// Call the controller
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123", map[string]interface{}{
				"adminRoleArn": "new:admin:role",
			}),
		)
		require.Nil(t, err)

		// Should return a 400
		require.Equal(t, 400, res.StatusCode)

		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"error": map[string]interface{}{
				"code": "RequestValidationError",
				"message": "Unable to update account 123: " +
					"admin role is not assumable by the master account",
			},
		}, resJSON)
	})

	t.Run("should not attempt to assume the adminRole, if none is provided", func(t *testing.T) {
		stubAllServices()
		tokenSvc := &commonMocks.TokenService{}
		cfgBldr := services.Config
		services = &config.ServiceBuilder{Config: cfgBldr}
		services.Config.WithService(&tokenSvc)

		// Call the controller with metadata update only
		_, err := Handler(context.TODO(),
			newUpdateRequest(t, "123", map[string]interface{}{
				"metadata": map[string]interface{}{},
			}),
		)
		require.Nil(t, err)

		// Check that we didn't assume role
		tokenSvc.AssertNumberOfCalls(t, "AssumeRole", 0)
	})

	t.Run("should fail if no updatable fields are provided", func(t *testing.T) {
		stubAllServices()

		// Call the controller with no updatable fields
		res, err := Handler(context.TODO(),
			newUpdateRequest(t, "123", map[string]interface{}{
				"accountStatus": "Ready",
			}),
		)
		require.Nil(t, err)

		require.Equal(t, 400, res.StatusCode)

		resJSON := unmarshal(t, res.Body)
		require.Equal(t, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "RequestValidationError",
				"message": "Unable to update account 123: no updatable fields provided",
			},
		}, resJSON)
	})

}

func newUpdateRequest(t *testing.T, accountID string, body map[string]interface{}) events.APIGatewayProxyRequest {
	return newRequest(t, "PUT", fmt.Sprintf("/accounts/%s", accountID), body)
}
