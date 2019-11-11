package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	"github.com/stretchr/testify/mock"
)

func TestDeleteController_Call(t *testing.T) {
	type fields struct {
		Dao                    db.DBer
		Queue                  common.Queue
		ResetQueueURL          string
		SNS                    common.Notificationer
		AccountDeletedTopicArn string
		AWSSession             session.Session
		TokenService           common.TokenService
		RoleManager            rolemanager.RoleManager
		PrincipalRoleName      string
		PrincipalPolicyName    string
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
	}

	// Set up the mocks...

	// A bad request. What this means in lease delete world is that we have failed to
	// parse the request body becausse it is empty.
	badArgs := &args{ctx: context.Background(), req: createBadDeleteRequest()}
	badRequestResponse := response.ClientBadRequestError(fmt.Sprintf("Failed to Parse Request Body: %s", ""))

	// Another bad request. There are no accounts for the principal that is in the lease
	// request
	noAccountsForLeaseArgs := &args{ctx: context.Background(), req: createNoAccountsForLeaseRequest()}
	mockDB.On("FindLeasesByPrincipal", "23456").Return(nil, nil)
	noAccountsForLeaseResponse := response.ClientBadRequestError("No leases found for 23456")

	// A client error, because there is no active account for the principal ID
	noActiveAccountForLeaseArgs := &args{ctx: context.Background(), req: createNoActiveAccountForLeaseRequest()}
	mockDB.On("FindLeasesByPrincipal", "67890").Return(createNonMatchingAccountListDBResponse(), nil)
	noActiveAccountForLeaseResponse := response.ClientBadRequestError("Lease is not active for 67890 - 987654321")

	// Successful delete
	successfulDeleteArgs := &args{ctx: context.Background(), req: createSuccessfulDeleteRequest()}
	mockDB.On("FindLeasesByPrincipal", "12345").Return(createSuccessfulDeleteDBResponse(), nil)
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
			c := DeleteController{
				Dao: tt.fields.Dao,
				SNS: tt.fields.SNS,
			}
			got, err := c.Call(tt.args.ctx, tt.args.req)
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

func createBadDeleteRequest() *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{}
}

func createNoAccountsForLeaseRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "23456",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createNoActiveAccountForLeaseRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "67890",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createSuccessfulDeleteRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "12345",
		AccountID:   "123456789",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createNoAccountsForLeaseDBResponse() []*db.Lease {
	leases := []*db.Lease{}
	return leases
}

func createNonMatchingAccountListDBResponse() []*db.Lease {
	leases := []*db.Lease{
		{
			AccountID:   "987654321",
			PrincipalID: "67890",
			LeaseStatus: db.Inactive,
		},
	}
	return leases
}

func createSuccessfulDeleteDBResponse() []*db.Lease {
	leases := []*db.Lease{
		{
			AccountID:   "123456789",
			PrincipalID: "12345",
			LeaseStatus: db.Active,
		},
	}
	return leases
}

func createAccountForDelete() *db.Account {
	return &db.Account{
		ID: "987654321",
	}
}

func createSuccessDeleteResponse() events.APIGatewayProxyResponse {
	lease := &db.Lease{
		PrincipalID: "12345",
		AccountID:   "123456789",
		LeaseStatus: db.Inactive,
	}
	leaseResponse := response.LeaseResponse(*lease)
	return response.CreateJSONResponse(http.StatusOK, leaseResponse)
}
