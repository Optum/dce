package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Optum/dce/pkg/api/response"
)

// GetUsageByStartDateAndEndDate - Returns a list of usage by startDate and endDate
func GetUsageByStartDateAndEndDate(w http.ResponseWriter, r *http.Request) {

	i, err := strconv.ParseInt(r.FormValue(StartDateParam), 10, 64)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to parse usage start date: %s", err)
		log.Println(errorMsg)
		response.WriteRequestValidationError(w, errorMsg)
		return
	}
	startDate := time.Unix(i, 0)

	j, err := strconv.ParseInt(r.FormValue(EndDateParam), 10, 64)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to parse usage end date: %s", err)
		log.Println(errorMsg)
		response.WriteRequestValidationError(w, errorMsg)
		return
	}
	endDate := time.Unix(j, 0)

	usageRecords, err := UsageSvc.GetUsageByDateRange(startDate, endDate)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting usage for given start date %s and end date %s: %s", r.FormValue(StartDateParam), r.FormValue(EndDateParam), err.Error())
		log.Println(errMsg)
		response.ServerErrorWithResponse(errMsg)
		return
	}

	// Serialize them for the JSON response.
	usageResponseItems := []*response.UsageResponse{}

	for _, a := range usageRecords {
		usageRes := response.UsageResponse(*a)
		usageRes.StartDate = startDate.Unix()
		usageRes.EndDate = endDate.Unix()
		log.Printf("usage: %v", usageRes)
		usageResponseItems = append(usageResponseItems, &usageRes)
	}

	outputResponseItems := SumCostAmountByPrincipalID(usageResponseItems)

	err = json.NewEncoder(w).Encode(outputResponseItems)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting usage for given start date %s and end date %s: %s", r.FormValue(StartDateParam), r.FormValue(EndDateParam), err.Error())
		log.Println(errMsg)
		response.ServerErrorWithResponse(errMsg)
		return
	}
}

// GetUsageByStartDateAndPrincipalID - Returns a list of usage by principalID and starting from start date to current date
func GetUsageByStartDateAndPrincipalID(w http.ResponseWriter, r *http.Request) {

	i, err := strconv.ParseInt(r.FormValue(StartDateParam), 10, 64)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to parse usage start date: %s", err)
		log.Println(errorMsg)
		response.WriteRequestValidationError(w, errorMsg)
		return
	}
	startDate := time.Unix(i, 0)

	principalID := r.FormValue(PrincipalIDParam)

	usageRecords, err := UsageSvc.GetUsageByPrincipal(startDate, principalID)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting usage for given start date %s and principalID %s: %s", r.FormValue(StartDateParam), principalID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}

	// Serialize them for the JSON response.
	usageResponseItems := []*response.UsageResponse{}

	for _, a := range usageRecords {
		usageRes := response.UsageResponse(*a)
		usageResponseItems = append(usageResponseItems, &usageRes)
	}

	err = json.NewEncoder(w).Encode(usageResponseItems)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting usage for given start date %s and principalID %s: %s", r.FormValue(StartDateParam), principalID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
}

// SumCostAmountByPrincipalID returns a unique subset of the input slice by finding unique PrincipalIds and adding cost amount for it.
func SumCostAmountByPrincipalID(input []*response.UsageResponse) []*response.UsageResponse {
	u := make([]*response.UsageResponse, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val.PrincipalID]; !ok {
			m[val.PrincipalID] = true
			u = append(u, val)
		} else {
			for i, item := range u {
				if item.PrincipalID == val.PrincipalID {
					log.Printf("item: %v", item)
					log.Printf("val: %v", val)
					u[i].CostAmount = u[i].CostAmount + val.CostAmount
					break
				}

			}
		}
	}

	return u
}
