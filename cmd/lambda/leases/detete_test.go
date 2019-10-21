package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	commonMock "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	mockDB "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/Optum/Redbox/pkg/rolemanager"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"
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

	lease := &db.RedboxLease{
		AccountID:   "987654321",
		PrincipalID: "67890",
		LeaseStatus: db.Inactive,
	}

	otherLease := &db.RedboxLease{
		AccountID:   "123456789",
		PrincipalID: "23456",
		LeaseStatus: db.Inactive,
	}

	// Set up the mocks...

	// A bad request. What this means in lease delete world is that we have failed to
	// parse the request body becausse it is empty.

	// Another bad request. There are no accounts for the principal that is in the lease
	// request

	// A client error, because there is no active account for the principal ID

	// Successful delete

	mockDB.On("FindLeasesByPrincipal", "12345").Return(createEmptyAccountListForDelete(), nil)
	mockDB.On("FindLeasesByPrincipal", "23456").Return(createNonMatchingAccountListForDelete(), nil)
	mockDB.On("FindLeasesByPrincipal", "67890").Return(createValidAccountListForDelete(), nil)
	mockDB.On("TransitionLeaseStatus", "987654321", "67890", db.Active, db.Inactive, db.LeaseDestroyed).Return(lease, nil)
	mockDB.On("TransitionLeaseStatus", "987654321", "23456", db.Active, db.Inactive, db.LeaseDestroyed).Return(otherLease, nil)
	mockDB.On("TransitionAccountStatus", "987654321", db.Leased, db.NotReady).Return(nil, nil)
	mockSNS.On("PublishMessage", &leaseTopicARN, mock.Anything, true).Return(&messageID, nil)

	testFields := &fields{
		Dao: mockDB,
		SNS: mockSNS,
	}

	successResponse := createSuccessDeleteResponse()
	badRequestResponse := response.ClientBadRequestError(fmt.Sprintf("Failed to Parse Request Body: %s", ""))
	notFoundRequest := response.NotFoundError()

	successArgs := &args{ctx: context.Background(), req: createSuccessfulDeleteRequest()}
	noMatchingAccountsArgs := &args{ctx: context.Background(), req: createNoMatchingLeasesRequest()}
	noExistingLeaseArgs := &args{ctx: context.Background(), req: createEmptyLeasesDeleteRequest()}
	badArgs := &args{ctx: context.Background(), req: createBadDeleteRequest()}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    events.APIGatewayProxyResponse
		wantErr bool
	}{
		{name: "Bad request.", fields: *testFields, args: *badArgs, want: badRequestResponse, wantErr: false},
		{name: "No matching leases.", fields: *testFields, args: *noExistingLeaseArgs, want: notFoundRequest, wantErr: false},
		{name: "No matching accounts.", fields: *testFields, args: *noMatchingAccountsArgs, want: notFoundRequest, wantErr: false},
		{name: "Successful delete.", fields: *testFields, args: *successArgs, want: successResponse, wantErr: false},
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

func createSuccessfulDeleteRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "67890",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createBadDeleteRequest() *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{}
}

func createEmptyLeasesDeleteRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "23456",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createNoMatchingLeasesRequest() *events.APIGatewayProxyRequest {
	deleteLeaseRequest := &deleteLeaseRequest{
		PrincipalID: "23456",
		AccountID:   "987654321",
	}
	requestBodyBytes, _ := json.Marshal(deleteLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createEmptyAccountListForDelete() []*db.RedboxLease {
	leases := []*db.RedboxLease{}
	return leases
}

func createValidAccountListForDelete() []*db.RedboxLease {
	leases := []*db.RedboxLease{
		{
			AccountID:   "987654321",
			PrincipalID: "67890",
			LeaseStatus: db.Active,
		},
	}
	return leases
}

func createNonMatchingAccountListForDelete() []*db.RedboxLease {
	leases := []*db.RedboxLease{
		{
			AccountID:   "987654321",
			PrincipalID: "67890",
			LeaseStatus: db.Active,
		},
	}
	return leases
}

func createAccountForDelete() *db.RedboxAccount {
	return &db.RedboxAccount{
		ID: "987654321",
	}
}

func createSuccessDeleteResponse() events.APIGatewayProxyResponse {
	lease := &db.RedboxLease{
		PrincipalID: "67890",
		AccountID:   "123456789",
		LeaseStatus: db.Inactive,
	}
	leaseResponse := response.LeaseResponse(*lease)
	return response.CreateJSONResponse(http.StatusOK, leaseResponse)
}
