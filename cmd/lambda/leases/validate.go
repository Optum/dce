package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/Optum/dce/pkg/usage"
)

const Weekly = "WEEKLY"
const Monthly = "MONTHLY"

type leaseValidationContext struct {
	maxLeaseBudgetAmount  float64
	principalBudgetAmount float64
	maxLeasePeriod        int64
	principalBudgetPeriod string
	usageRecords          []*usage.Usage
}

// ValidateLease validates lease budget amount and period
func validateLeaseFromRequest(context *leaseValidationContext, req *http.Request) (*createLeaseRequest, bool, string, error) {

	// Validate body from the Request
	requestBody := &createLeaseRequest{}
	var err error

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&requestBody)

	if err != nil || requestBody.PrincipalID == "" {
		validationErrStr := fmt.Sprintf("Failed to Parse Request Body: %s", req.Body)
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease end date is greater than today
	if requestBody.ExpiresOn != 0 && requestBody.ExpiresOn <= time.Now().Unix() {
		validationErrStr := fmt.Sprintf("Requested lease has a desired expiry date less than today: %d", requestBody.ExpiresOn)
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease budget amount is less than MAX_LEASE_BUDGET_AMOUNT
	if requestBody.BudgetAmount > context.maxLeaseBudgetAmount {
		validationErrStr := fmt.Sprintf("Requested lease has a budget amount of %f, which is greater than max lease budget amount of %f", math.Round(requestBody.BudgetAmount), math.Round(context.maxLeaseBudgetAmount))
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease budget period is less than MAX_LEASE_BUDGET_PERIOD
	currentTime := time.Now()
	maxLeaseExpiresOn := currentTime.Add(time.Second * time.Duration(context.maxLeasePeriod))
	if requestBody.ExpiresOn > maxLeaseExpiresOn.Unix() {
		validationErrStr := fmt.Sprintf("Requested lease has a budget expires on of %d, which is greater than max lease period of %d", requestBody.ExpiresOn, maxLeaseExpiresOn.Unix())
		return requestBody, false, validationErrStr, nil
	}

	// Validate requested lease budget amount is less than PRINCIPAL_BUDGET_AMOUNT for current principal billing period
	// usageStartTime := getBeginningOfCurrentBillingPeriod(context.principalBudgetPeriod)
	// usageEndTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, time.UTC)

	// usageRecords, err := UsageSvc.GetUsageByDateRange(usageStartTime, usageEndTime)
	// if err != nil {
	// 	errStr := fmt.Sprintf("Failed to retrieve usage: %s", err)
	// 	return requestBody, true, "", errors.New(errStr)
	// }

	// Group by PrincipalID to get sum of total spent for current billing period
	spent := 0.0
	for _, usageItem := range context.usageRecords {
		if usageItem.PrincipalID == requestBody.PrincipalID {
			spent = spent + usageItem.CostAmount
		}
	}

	if spent > context.principalBudgetAmount {
		validationErrStr := fmt.Sprintf("Unable to create lease: User principal %s has already spent %f of their principal budget", requestBody.PrincipalID, math.Round(context.principalBudgetAmount))
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
