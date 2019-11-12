package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	commonMock "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	mockDB "github.com/Optum/dce/pkg/db/mocks"
	"github.com/Optum/dce/pkg/provision"
	provisionMock "github.com/Optum/dce/pkg/provision/mocks"
	"github.com/Optum/dce/pkg/usage"
	mockUsage "github.com/Optum/dce/pkg/usage/mocks"
	"github.com/aws/aws-lambda-go/events"
)

func TestCreateController_Call(t *testing.T) {

	Config = common.DefaultEnvConfig{}

	type (
		fields struct {
			Dao                   db.DBer
			Provisioner           provision.Provisioner
			SNS                   common.Notificationer
			LeaseTopicARN         *string
			UsageSvc              usage.Service
			PrincipalBudgetAmount *float64
			PrincipalBudgetPeriod *string
			MaxLeaseBudgetAmount  *float64
			MaxLeasePeriod        *int
		}
	)
	type args struct {
		ctx context.Context
		req *events.APIGatewayProxyRequest
	}

	leaseTopicARN := "some-topic-arn"
	messageID := "message123456789"

	principalBudgetAmount := 1000.00
	principalBudgetPeriod := "WEEKLY"
	maxLeaseBudgetAmount := 1000.00
	MaxLeasePeriod := 704800

	mockDB := &mockDB.DBer{}
	mockProv := &provisionMock.Provisioner{}
	mockSNS := &commonMock.Notificationer{}
	mockUsage := &mockUsage.Service{}

	// Set up the mocks...
	mockProv.On("FindActiveLeaseForPrincipal", "123456").Return(createActiveLease(), nil)
	mockDB.On("GetReadyAccount").Return(createAccount(), nil)
	mockProv.On("FindLeaseWithAccount", "123456", "987654321").Return(createActiveLease(), nil)
	mockProv.On("ActivateAccount",
		true, "123456", "987654321", float64(50), "USD", mock.Anything, mock.Anything).Return(createActiveLease(), nil)
	mockDB.On("TransitionAccountStatus", "987654321", db.Ready, db.Leased).Return(createAccount(), nil)
	mockSNS.On("PublishMessage", &leaseTopicARN, mock.Anything, true).Return(&messageID, nil)
	mockUsage.On("GetUsageByDateRange", mock.Anything, mock.Anything).Return(nil, nil)

	testFields := &fields{
		Dao:                   mockDB,
		Provisioner:           mockProv,
		SNS:                   mockSNS,
		LeaseTopicARN:         &leaseTopicARN,
		UsageSvc:              mockUsage,
		PrincipalBudgetAmount: &principalBudgetAmount,
		PrincipalBudgetPeriod: &principalBudgetPeriod,
		MaxLeaseBudgetAmount:  &maxLeaseBudgetAmount,
		MaxLeasePeriod:        &MaxLeasePeriod,
	}

	successResponse := createSuccessCreateResponse()
	badRequestResponse := response.ClientBadRequestError(fmt.Sprintf("Failed to Parse Request Body: %s", ""))
	pastRequestResponse := response.BadRequestError("Requested lease has a desired expiry date less than today: 1570627876")
	invalidBudgetRequestResponse := response.BadRequestError("Requested lease has a budget amount of 5000.000000, which is greater than max lease budget amount of 1000.000000")
	invalidBudgetPeriodRequestResponse := response.BadRequestError("Requested lease has a budget expires on of 1577745392, which is greater than max lease period of 1573176192")

	successArgs := &args{ctx: context.Background(), req: createSuccessfulCreateRequest()}
	pastArgs := &args{ctx: context.Background(), req: createPastCreateRequest()}
	invalidBudgetArgs := &args{ctx: context.Background(), req: invalidBudgetAmountCreateRequest()}
	invalidBudgetPeriodArgs := &args{ctx: context.Background(), req: invalidBudgetPeriodCreateRequest()}
	badArgs := &args{ctx: context.Background(), req: createBadCreateRequest()}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    events.APIGatewayProxyResponse
		wantErr bool
	}{
		{name: "Bad request.", fields: *testFields, args: *badArgs, want: badRequestResponse, wantErr: false},
		{name: "Past request.", fields: *testFields, args: *pastArgs, want: pastRequestResponse, wantErr: false},
		{name: "Invalid budget amount request.", fields: *testFields, args: *invalidBudgetArgs, want: invalidBudgetRequestResponse, wantErr: false},
		{name: "Invalid budget period request.", fields: *testFields, args: *invalidBudgetPeriodArgs, want: invalidBudgetPeriodRequestResponse, wantErr: false},
		{name: "Successful create.", fields: *testFields, args: *successArgs, want: *successResponse, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := Handler(tt.args.ctx, *tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {

				//comparing first 50 characters of error message for invalid budget period error
				if strings.HasPrefix(tt.want.Body, got.Body[:30]) {
					return
				}
				t.Errorf("Handler() = %v, want %v", got, tt.want)
			}
		})
	}

}

func createSuccessfulCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "123456",
		AccountID:                "987654321",
		BudgetAmount:             50,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                time.Now().AddDate(0, 0, 7).Unix(),
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createBadCreateRequest() *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{}
}

func createPastCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "123456",
		AccountID:                "987654321",
		BudgetAmount:             50,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                1570627876,
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func invalidBudgetAmountCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "123456",
		AccountID:                "987654321",
		BudgetAmount:             5000,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                time.Now().AddDate(0, 0, 5).Unix(),
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func invalidBudgetPeriodCreateRequest() *events.APIGatewayProxyRequest {
	createLeaseRequest := &createLeaseRequest{
		PrincipalID:              "123456",
		AccountID:                "987654321",
		BudgetAmount:             50,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"user3@example.com", "user2@example.com"},
		ExpiresOn:                time.Now().AddDate(0, 2, 0).Unix(),
	}
	requestBodyBytes, _ := json.Marshal(createLeaseRequest)
	return &events.APIGatewayProxyRequest{
		Body: string(requestBodyBytes),
	}
}

func createActiveLease() *db.Lease {
	return &db.Lease{}
}

func createAccount() *db.Account {
	return &db.Account{
		ID: "987654321",
	}
}

func createSuccessCreateResponse() *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		StatusCode: 201,
		Body:       "{\"accountId\":\"\",\"principalId\":\"\",\"id\":\"\",\"leaseStatus\":\"\",\"leaseStatusReason\":\"\",\"createdOn\":0,\"lastModifiedOn\":0,\"budgetAmount\":0,\"budgetCurrency\":\"\",\"budgetNotificationEmails\":null,\"leaseStatusModifiedOn\":0,\"expiresOn\":0}",
	}
}
