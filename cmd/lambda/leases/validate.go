package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
	"math"
	"time"
)

const Weekly = "WEEKLY"
const Monthly = "MONTHLY"

// validateLeaseRequest validates lease budget amount and period
func validateLeaseRequest(controller CreateController, req *events.APIGatewayProxyRequest) (*createLeaseRequest, bool, string, error) {

	// Validate body from the Request
	requestBody := &createLeaseRequest{}
	var err error

	err = json.Unmarshal([]byte(req.Body), requestBody)
	if err != nil || requestBody.PrincipalID == "" {
		validationErrStr := "invalid request parameters"
		return requestBody, false, validationErrStr, nil
	}

	// Set default expiresOn
	if requestBody.ExpiresOn == 0 {
		requestBody.ExpiresOn = time.Now().AddDate(0, 0, controller.DefaultLeaseLengthInDays).Unix()
	}

	// Set default metadata (empty object)
	if requestBody.Metadata == nil {
		requestBody.Metadata = map[string]interface{}{}
	}

	// Validate requested lease end date is greater than today
	if requestBody.ExpiresOn <= time.Now().Unix() {
		validationErrStr := fmt.Sprintf("Requested lease has a desired expiry date less than today: %d", requestBody.ExpiresOn)
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease budget amount is less than MAX_LEASE_BUDGET_AMOUNT
	if requestBody.BudgetAmount > *controller.MaxLeaseBudgetAmount {
		validationErrStr := fmt.Sprintf("Requested lease has a budget amount of %f, which is greater than max lease budget amount of %f", math.Round(requestBody.BudgetAmount), math.Round(*controller.MaxLeaseBudgetAmount))
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease budget period is less than MAX_LEASE_BUDGET_PERIOD
	currentTime := time.Now()
	maxLeaseExpiresOn := currentTime.Add(time.Second * time.Duration(*controller.MaxLeasePeriod))
	if requestBody.ExpiresOn > maxLeaseExpiresOn.Unix() {
		validationErrStr := fmt.Sprintf("Requested lease has a budget expires on of %d, which is greater than max lease period of %d", requestBody.ExpiresOn, maxLeaseExpiresOn.Unix())
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease budget amount is less than PRINCIPAL_BUDGET_AMOUNT for current principal billing period
	usageStartTime := getBeginningOfCurrentBillingPeriod(*controller.PrincipalBudgetPeriod)

	usageRecords, err := controller.UsageSvc.GetUsageByPrincipal(usageStartTime, requestBody.PrincipalID)
	if err != nil {
		errStr := fmt.Sprintf("Failed to retrieve usage: %s", err)
		return requestBody, true, "", errors.New(errStr)
	}

	// Group by PrincipalID to get sum of total spent for current billing period
	spent := 0.0
	for _, usageItem := range usageRecords {
		spent = spent + usageItem.CostAmount
	}

	if spent > *controller.PrincipalBudgetAmount {
		validationErrStr := fmt.Sprintf("Unable to create lease: User principal %s has already spent %f of their principal budget", requestBody.PrincipalID, math.Round(*controller.PrincipalBudgetAmount))
		return requestBody, false, validationErrStr, nil
	}

	return requestBody, true, "", nil
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
