package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Optum/Redbox/pkg/api"
	apiMocks "github.com/Optum/Redbox/pkg/api/mocks"
	commonMocks "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/require"
)

func TestGetLeaseAuth(t *testing.T) {

	t.Run("When the invoking Call and there are no errors", func(t *testing.T) {

		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			require.Equal(t, req.URL.Path, "/federation")
			q, _ := url.ParseQuery(req.URL.RawQuery)
			require.Equal(t, q.Get("Action"), "getSigninToken")
			require.Equal(t, q.Get("Session"), `{"sessionId":"ExampleKey","sessionKey":"ExampleSecret","sessionToken":"ExampleSession"}`)
			fmt.Fprintf(rw, `{"SigninToken":"ExampleSigninToken"}`)
		}))

		federationURL := fmt.Sprintf("%s/federation", server.URL)
		consoleURL := fmt.Sprintf("%s/console", server.URL)

		tests := []struct {
			name             string
			ctx              context.Context
			expectedResponse *events.APIGatewayProxyResponse
			expectedErr      error
			principalRoleArn string
			accountID        string
			leaseID          string
			getLeaseByIDErr  error
			getAccountErr    error
			assumeRoleErr    error
			leaseStatus      db.LeaseStatus
			userName         string
			userRole         string
		}{
			{
				name:      "WorkingPath",
				accountID: "Account123",
				leaseID:   "LeaseABC",
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 201,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: fmt.Sprintf(
						`{"accessKeyId":"ExampleKey","secretAccessKey":"ExampleSecret","sessionToken":"ExampleSession","consoleUrl":"%s"}`,
						fmt.Sprintf(
							`%s?Action=login\u0026Destination=%s\u0026Issuer=DCE\u0026SigninToken=ExampleSigninToken`,
							federationURL,
							url.QueryEscape(consoleURL)),
					),
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "LeaseNotFound",
				getLeaseByIDErr: nil,
				getAccountErr:   nil,
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 404,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"NotFound","message":"The requested resource could not be found."}}`,
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "GetLeaseError",
				leaseID:         "Lease987",
				getLeaseByIDErr: fmt.Errorf("Get Lease Error"),
				getAccountErr:   nil,
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 500,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"ServerError","message":"Failed Get on Lease Lease987"}}`,
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "AccountNotFound",
				leaseID:         "Lease123",
				getLeaseByIDErr: nil,
				getAccountErr:   nil,
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 500,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"ServerError","message":"Account  could not be found"}}`,
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "GetAccountError",
				leaseID:         "Lease987",
				accountID:       "Account987",
				getLeaseByIDErr: nil,
				getAccountErr:   fmt.Errorf("Get Lease Error"),
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 500,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"ServerError","message":"Failed List on Account Account987"}}`,
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "GetAssumeRoleError",
				leaseID:         "Lease987",
				accountID:       "Account987",
				getLeaseByIDErr: nil,
				getAccountErr:   nil,
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 500,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"ServerError","message":"Internal server error"}}`,
				},
				assumeRoleErr:    fmt.Errorf("Token Error"),
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "LeaseInactive",
				leaseID:         "Lease987",
				accountID:       "Account987",
				getLeaseByIDErr: nil,
				getAccountErr:   nil,
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 401,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"Unauthorized","message":"Could not access the resource requested."}}`,
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Inactive,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.AdminGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
			{
				name:            "UserHasNoAccessToLease",
				leaseID:         "Lease987",
				accountID:       "Account987",
				getLeaseByIDErr: nil,
				getAccountErr:   nil,
				expectedResponse: &events.APIGatewayProxyResponse{
					StatusCode: 404,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: `{"error":{"code":"NotFound","message":"The requested resource could not be found."}}`,
				},
				assumeRoleErr:    nil,
				leaseStatus:      db.Active,
				expectedErr:      nil,
				userName:         "TestUser",
				userRole:         api.UserGroupName,
				principalRoleArn: "arn:aws:iam::Account123:role/Principal",
			},
		}

		// Close the server when test finishes
		defer server.Close()

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expectedLease := &db.Lease{}
				expectedAccount := &db.Account{}

				mockRequest := events.APIGatewayProxyRequest{
					HTTPMethod: http.MethodGet,
					Path:       "/leases/badLease/auth",
					PathParameters: map[string]string{
						"id": "badLease",
					},
					RequestContext: events.APIGatewayProxyRequestContext{
						Identity: events.APIGatewayRequestIdentity{
							CognitoIdentityPoolID:         "us_east_1-test",
							CognitoAuthenticationProvider: fmt.Sprintf("UserPoolID:CognitoSignIn:%s", tt.userName),
						},
					},
				}
				mockDb := mocks.DBer{}

				if tt.leaseID != "" {
					expectedLease = &db.Lease{
						ID:          tt.leaseID,
						AccountID:   tt.accountID,
						LeaseStatus: tt.leaseStatus,
					}
					mockRequest = events.APIGatewayProxyRequest{
						HTTPMethod: http.MethodGet,
						Path:       fmt.Sprintf("/leases/%s/auth", tt.leaseID),
						PathParameters: map[string]string{
							"id": tt.leaseID,
						},
					}
					mockDb.On("GetLeaseByID", tt.leaseID).Return(expectedLease, tt.getLeaseByIDErr)
				} else {
					mockDb.On("GetLeaseByID", "badLease").Return(nil, tt.getLeaseByIDErr)
				}
				if tt.accountID != "" {
					expectedAccount = &db.Account{
						ID:               tt.accountID,
						AccountStatus:    db.Ready,
						PrincipalRoleArn: tt.principalRoleArn,
					}
					mockDb.On("GetAccount", tt.accountID).Return(expectedAccount, tt.getAccountErr)
				} else {
					mockDb.On("GetAccount", "").Return(nil, tt.getAccountErr)
				}

				mockToken := commonMocks.TokenService{}
				mockToken.On("AssumeRole",
					&sts.AssumeRoleInput{
						RoleArn:         aws.String(tt.principalRoleArn),
						RoleSessionName: aws.String(tt.userName),
					},
				).Return(
					&sts.AssumeRoleOutput{
						Credentials: &sts.Credentials{
							AccessKeyId:     aws.String("ExampleKey"),
							SecretAccessKey: aws.String("ExampleSecret"),
							SessionToken:    aws.String("ExampleSession"),
						},
					}, tt.assumeRoleErr,
				)

				mockUserDetailer := apiMocks.UserDetailer{}
				mockUserDetailer.On("GetUser", &mockRequest).Return(&api.User{
					Role:     tt.userRole,
					Username: tt.userName,
				})

				controller := CreateController{
					Dao:           &mockDb,
					TokenService:  &mockToken,
					ConsoleURL:    consoleURL,
					FederationURL: federationURL,
					UserDetailer:  &mockUserDetailer,
				}

				actualResponse, err := controller.Call(context.TODO(), &mockRequest)
				require.Nil(t, err)
				require.Equal(t, *tt.expectedResponse, actualResponse, "Response matches")
			})
		}

	})

}
