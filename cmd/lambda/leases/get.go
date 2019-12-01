package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/Optum/dce/pkg/db"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

type GetController struct {
	Dao db.DBer
}

// Call - function to return a specific AWS Lease record to the request
func (controller GetController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the account.
	leaseID := path.Base(req.Path)
	lease, err := controller.Dao.GetLeaseByID(leaseID)
	if err != nil {
		log.Printf("Error Getting Lease for Id: %s", leaseID)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed Get on Lease %s",
					leaseID))), nil
	}
	if lease == nil {
		log.Printf("Error Getting Lease for Id: %s", err)
		return response.NotFoundError(), nil
	}

	leaseResponse := response.LeaseResponse(*lease)
	return response.CreateApiGatewayJSONResponse(http.StatusOK, leaseResponse), nil
}
