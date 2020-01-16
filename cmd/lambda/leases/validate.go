package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

type leaseValidationContext struct {
	maxLeaseBudgetAmount     float64
	principalBudgetAmount    float64
	maxLeasePeriod           int64
	principalBudgetPeriod    string
	defaultLeaseLengthInDays int
}

// ValidateLease validates lease budget amount and period
func validateLeaseFromRequest(context *leaseValidationContext, req *http.Request) (*createLeaseRequest, bool, string, error) {

	// Validate body from the Request
	requestBody := &createLeaseRequest{}
	var err error

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&requestBody)

	if err != nil || requestBody.PrincipalID == "" {
		validationErrStr := "invalid request parameters"
		return requestBody, false, validationErrStr, nil
	}

	// Set default expiresOn
	if requestBody.ExpiresOn == 0 {
		requestBody.ExpiresOn = time.Now().AddDate(0, 0, context.defaultLeaseLengthInDays).Unix()
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
	usageStartTime := getBeginningOfCurrentBillingPeriod(context.principalBudgetPeriod)

	usageRecords, err := usageSvc.GetUsageByPrincipal(usageStartTime, requestBody.PrincipalID)
	if err != nil {
		errStr := fmt.Sprintf("Failed to retrieve usage: %s", err)
		return requestBody, true, "", errors.New(errStr)
	}

	// Group by PrincipalID to get sum of total spent for current billing period
	spent := 0.0
	for _, usageItem := range usageRecords {
		spent = spent + usageItem.CostAmount
	}

	if spent > context.principalBudgetAmount {
		validationErrStr := fmt.Sprintf(
			"Unable to create lease: User principal %s has already spent %.2f of their %.2f principal budget",
			requestBody.PrincipalID, spent, context.principalBudgetAmount,
		)
		return requestBody, false, validationErrStr, nil
	}

	return requestBody, true, "", nil
}
