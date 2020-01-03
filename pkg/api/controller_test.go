package api_test

import (
	"context"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/Optum/dce/pkg/api"
	mockController "github.com/Optum/dce/pkg/api/mocks"
	"github.com/aws/aws-lambda-go/events"
)

func TestRouter_Route(t *testing.T) {
	mockListController := &mockController.Controller{}
	mockGetController := &mockController.Controller{}
	mockDeleteController := &mockController.Controller{}
	mockCreateController := &mockController.Controller{}
	mockUserDetails := &mockController.UserDetailer{}

	ctx := context.Background()

	listLeasesRequest := &events.APIGatewayProxyRequest{
		Path:       "/leases",
		HTTPMethod: "GET",
	}

	getLeaseRequest := &events.APIGatewayProxyRequest{
		Path:       "/leases/34232342",
		HTTPMethod: "GET",
	}

	createLeaseRequest := &events.APIGatewayProxyRequest{
		Path:       "/leases",
		HTTPMethod: "POST",
	}

	deleteLeaseRequest := &events.APIGatewayProxyRequest{
		Path:       "/leases/",
		HTTPMethod: "DELETE",
	}

	router := &api.Router{
		ResourceName:     "/leases",
		CreateController: mockCreateController,
		ListController:   mockListController,
		GetController:    mockGetController,
		DeleteController: mockDeleteController,
		UserDetails:      mockUserDetails,
	}

	tests := []struct {
		name               string
		request            events.APIGatewayProxyRequest
		ctx                context.Context
		expectedController *mockController.Controller
		expectedErr        error
		user               api.User
	}{
		{
			name:               "GET (list) HTTP...",
			request:            *listLeasesRequest,
			ctx:                ctx,
			expectedController: mockListController,
			user: api.User{
				Role: api.AdminGroupName,
			},
		},
		{
			name:               "GET (single) HTTP...",
			request:            *getLeaseRequest,
			ctx:                ctx,
			expectedController: mockGetController,
			user: api.User{
				Role: api.AdminGroupName,
			},
		},
		{
			name:               "DELETE HTTP...",
			request:            *deleteLeaseRequest,
			ctx:                ctx,
			expectedController: mockDeleteController,
			user: api.User{
				Role: api.AdminGroupName,
			},
		},
		{
			name:               "POST (create) HTTP...",
			request:            *createLeaseRequest,
			ctx:                ctx,
			expectedController: mockCreateController,
			user: api.User{
				Role: api.AdminGroupName,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &events.APIGatewayProxyResponse{}
			ctxWIthUser := context.WithValue(tt.ctx, api.DceCtxKey, tt.user)
			tt.expectedController.On("Call", ctxWIthUser, &tt.request).Return(*res, nil)
			mockUserDetails.On("GetUser", mock.Anything).Return(&tt.user)
			_, _ = router.Route(tt.ctx, &tt.request)
			tt.expectedController.AssertExpectations(t)
		})
	}

}
