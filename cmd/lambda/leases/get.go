package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/Optum/Redbox/pkg/db"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

type getController struct {
	Dao db.DBer
}

// Call - function to return a specific AWS Lease record to the request
func (controller getController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Fetch the account.
	leaseID := path.Base(req.Path)
	lease, err := controller.Dao.GetLeaseByID(leaseID)
	if err != nil {
		log.Printf("Error Getting Lease for Id: %s", err)
		return response.CreateAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed Get on Lease %s",
					leaseID))), nil
	}
	if lease == nil {
		log.Printf("Error Getting Lease for Id: %s", err)
		return response.NotFoundError(), nil
	}

	leaseResponse := response.LeaseResponse(*lease)
	return response.CreateJSONResponse(http.StatusOK, leaseResponse), nil
}
