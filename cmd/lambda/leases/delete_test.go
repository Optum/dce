package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	commonMock "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	mockDB "github.com/Optum/dce/pkg/db/mocks"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteController_Call(t *testing.T) {
	type fields struct {
		Dao                 db.DBer
		Queue               common.Queue
		SNS                 common.Notificationer
		AWSSession          session.Session
		TokenService        common.TokenService
		RoleManager         rolemanager.RoleManager
		PrincipalRoleName   string
		PrincipalPolicyName string
	}
	type args struct {
		ctx context.Context
		req *events.APIGatewayProxyRequest
	}

	leaseTopicARN := "some-topic-arn"
	messageID := "message123456789"

	mockDB := &mockDB.DBer{}
	mockSNS := &commonMock.Notificationer{}

	lease := &db.Lease{
		AccountID:   "123456789",
		PrincipalID: "12345",
		LeaseStatus: db.Inactive,
		ID:          "abc",
	}

	// Set up the mocks...

	// A bad request. What this means in lease delete world is that we have failed to
	// parse the request body becausse it is empty.
	badArgs := &args{ctx: context.Background(), req: createBadDeleteRequest()}
	badRequestResponse := response.CreateMultiValueHeaderAPIErrorResponse(http.StatusBadRequest, "ClientError", fmt.Sprintf("Failed to Parse Request Body: %s", "{}"))

	// Another bad request. There are no accounts for the principal that is in the lease
	// request
	noAccountsForLeaseArgs := &args{ctx: context.Background(), req: createNoAccountsForLeaseRequest()}
	mockDB.On("GetLease", "987654321", "23456").Return(nil, nil)
	noAccountsForLeaseResponse := response.CreateMultiValueHeaderAPIErrorResponse(http.StatusBadRequest, "ClientError", "No leases found for Principal \"23456\" and Account ID \"987654321\"")

	// A client error, because there is no active account for the principal ID
	noActiveAccountForLeaseArgs := &args{ctx: context.Background(), req: createNoActiveAccountForLeaseRequest()}
	mockDB.On("GetLease", "987654321", "67890").Return(createNonMatchingAccountListDBResponse(), nil)
	noActiveAccountForLeaseResponse := response.CreateMultiValueHeaderAPIErrorResponse(http.StatusBadRequest, "ClientError", "Lease is not active for \"abc\"")

	// Successful delete
	successfulDeleteArgs := &args{ctx: context.Background(), req: createSuccessfulDeleteRequest()}
	mockDB.On("GetLease", "123456789", "12345").Return(createSuccessfulDeleteDBResponse(), nil)
	mockDB.On("TransitionLeaseStatus", "123456789", "12345", db.Active, db.Inactive, db.LeaseDestroyed).Return(lease, nil)
	mockDB.On("TransitionAccountStatus", "123456789", db.Leased, db.NotReady).Return(nil, nil)
	mockSNS.On("PublishMessage", &leaseTopicARN, mock.Anything, true).Return(&messageID, nil)
	successResponse := createSuccessDeleteResponse()

	testFields := &fields{
		Dao: mockDB,
		SNS: mockSNS,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    events.APIGatewayProxyResponse
		wantErr bool
	}{
		{name: "Bad request.", fields: *testFields, args: *badArgs, want: badRequestResponse, wantErr: false},
		{name: "No matching accounts.", fields: *testFields, args: *noAccountsForLeaseArgs, want: noAccountsForLeaseResponse, wantErr: false},
		{name: "No matching leases.", fields: *testFields, args: *noActiveAccountForLeaseArgs, want: noActiveAccountForLeaseResponse, wantErr: false},
		{name: "Successful delete.", fields: *testFields, args: *successfulDeleteArgs, want: successResponse, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			dao = tt.fields.Dao

			got, err := Handler(tt.args.ctx, *tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteController.Call() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeleteController.Call() = %v, want %v", got, tt.want)
			}
		})
	}

}

func TestDeleteByID(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name               string
		expResp            response
		leaseID            string
		getLease           *db.Lease
		getErr             error
		transitiionedLease *db.Lease
		transitionErr      error
	}{
		{
			name:    "successful delete",
			leaseID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{\"accountId\":\"123456789012\",\"principalId\":\"principal\",\"id\":\"abc123\",\"leaseStatus\":\"Inactive\",\"leaseStatusReason\":\"\",\"createdOn\":0,\"lastModifiedOn\":0,\"budgetAmount\":0,\"budgetCurrency\":\"\",\"budgetNotificationEmails\":null,\"leaseStatusModifiedOn\":0,\"expiresOn\":0,\"metadata\":null}\n",
			},
			getLease: &db.Lease{
				ID:          "abc123",
				LeaseStatus: db.Active,
				PrincipalID: "principal",
				AccountID:   "123456789012",
			},
			transitiionedLease: &db.Lease{
				ID:          "abc123",
				LeaseStatus: db.Inactive,
				PrincipalID: "principal",
				AccountID:   "123456789012",
			},
			getErr: nil,
		},
		{
			name:    "failure no lease by that ID",
			leaseID: "abc123",
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"code\":\"ServerError\",\"message\":\"Cannot verify if Lease ID \\\"abc123\\\" exists\"}}",
			},
			getLease: &db.Lease{
				ID:          "abc123",
				LeaseStatus: db.Active,
				PrincipalID: "principal",
				AccountID:   "123456789012",
			},
			transitiionedLease: &db.Lease{
				ID:          "abc123",
				LeaseStatus: db.Inactive,
				PrincipalID: "principal",
				AccountID:   "123456789012",
			},
			getErr: fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("DELETE", fmt.Sprintf("http://example.com/lease/%s", tt.leaseID), nil)

			r = mux.SetURLVars(r, map[string]string{
				"leaseID": tt.leaseID,
			})
			w := httptest.NewRecorder()

			mockDB := &mockDB.DBer{}
			mockDB.On("GetLeaseByID", tt.leaseID).Return(tt.getLease, tt.getErr)
			mockDB.On("TransitionLeaseStatus",
				tt.getLease.AccountID,
				tt.getLease.PrincipalID, db.Active,
				db.Inactive, db.LeaseDestroyed).
				Return(tt.transitiionedLease, nil)
			mockDB.On("TransitionAccountStatus",
				tt.getLease.AccountID,
				db.Leased, db.NotReady).
				Return(nil, nil)
			mockSNS := &commonMock.Notificationer{}

			dao = mockDB
			snsSvc = mockSNS
			DeleteLeaseByID(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}

}

func createBadDeleteRequest() *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodDelete,
		Path:       "/leases",
	}
}

func createNoAccountsForLeaseRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "23456",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodDelete,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

func createNoActiveAccountForLeaseRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "67890",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodDelete,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

func createSuccessfulDeleteRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "12345",
		AccountID:   "123456789",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodDelete,
		Path:       "/leases",
		Body:       string(requestBodyBytes),
	}
}

func createNonMatchingAccountListDBResponse() *db.Lease {
	return &db.Lease{
		AccountID:   "987654321",
		PrincipalID: "67890",
		LeaseStatus: db.Inactive,
		ID:          "abc",
	}
}

func createSuccessfulDeleteDBResponse() *db.Lease {
	return &db.Lease{
		AccountID:   "123456789",
		PrincipalID: "12345",
		LeaseStatus: db.Active,
		ID:          "abc",
	}
}

func createSuccessDeleteResponse() events.APIGatewayProxyResponse {
	lease := &db.Lease{
		PrincipalID: "12345",
		AccountID:   "123456789",
		LeaseStatus: db.Inactive,
		ID:          "abc",
	}
	leaseResponse := response.LeaseResponse(*lease)
	return response.CreateMultiValueHeaderJSONResponse(http.StatusOK, leaseResponse)
}
