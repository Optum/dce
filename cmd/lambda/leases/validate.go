package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"math"
	"time"
)

const Weekly = "WEEKLY"
const Monthly = "MONTHLY"

// validateLeaseRequest validates lease budget amount and period
func validateLeaseRequest(controller CreateController, req *events.APIGatewayProxyRequest) (*createLeaseRequest, error) {

	// Validate body from the Request
	requestBody := &createLeaseRequest{}
	var err error
	if req.HTTPMethod != "GET" {
		err = json.Unmarshal([]byte(req.Body), requestBody)
		if err != nil || requestBody.PrincipalID == "" {
			errStr := fmt.Sprintf("Failed to Parse Request Body: %s", req.Body)
			return requestBody, errors.New(errStr)
		}
	}

	// Validate requested lease end date is greater than today
	if requestBody.ExpiresOn != 0 && requestBody.ExpiresOn <= time.Now().Unix() {
		errStr := fmt.Sprintf("Requested lease has a desired expiry date less than today: %d", requestBody.ExpiresOn)
		return requestBody, errors.New(errStr)
	}

	// Validate requested lease budget amount is less than MAX_LEASE_BUDGET_AMOUNT
	if requestBody.BudgetAmount > *controller.MaxLeaseBudgetAmount {
		errStr := fmt.Sprintf("Requested lease has a budget amount of %f, which is greater than max lease budget amount of %f", math.Round(requestBody.BudgetAmount), math.Round(*controller.MaxLeaseBudgetAmount))
		return requestBody, errors.New(errStr)
	}

	// Validate requested lease budget period is less than MAX_LEASE_BUDGET_PERIOD
	currentTime := time.Now()
	maxLeaseExpiresOn := currentTime.Add(time.Second * time.Duration(*controller.MaxLeasePeriod))
	if requestBody.ExpiresOn > maxLeaseExpiresOn.Unix() {
		errStr := fmt.Sprintf("Requested lease has a budget expires on of %d, which is greater than max lease period of %d", requestBody.ExpiresOn, maxLeaseExpiresOn.Unix())
		return requestBody, errors.New(errStr)
	}

	// Validate requested lease budget amount is less than PRINCIPAL_BUDGET_AMOUNT for current principal billing period
	usageStartTime := getBeginningOfCurrentBillingPeriod(*controller.PrincipalBudgetPeriod)
	usageEndTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, time.UTC)

	usageRecords, err := controller.UsageSvc.GetUsageByDateRange(usageStartTime, usageEndTime)
	if err != nil {
		errStr := fmt.Sprintf("Failed to retrieve usage: %s", err)
		return requestBody, errors.New(errStr)
	}

	// Group by PrincipalID to get sum of total spent for current billing period
	spent := 0.0
	for _, usageItem := range usageRecords {
		if usageItem.PrincipalID == requestBody.PrincipalID {
			spent = spent + usageItem.CostAmount
		}
	}

	if spent > *controller.PrincipalBudgetAmount {
		errStr := fmt.Sprintf("Unable to create lease: User principal %s has already spent %f of their weekly principal budget", requestBody.PrincipalID, math.Round(*controller.PrincipalBudgetAmount))
		return requestBody, errors.New(errStr)
	}

	return requestBody, nil
}

// getBeginningOfCurrentBillingPeriod returns starts of the billing period based on budget period
func getBeginningOfCurrentBillingPeriod(input string) time.Time {
	currentTime := time.Now()
	if input == Weekly {

		for currentTime.Weekday() != time.Sunday { // iterate back to Sunday
			currentTime = currentTime.AddDate(0, 0, -1)
		}

		return time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
	}

	return time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC)
}
