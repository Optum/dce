package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/Redbox/pkg/api/response"
)

// GetLease - Gets a lease by ID
func GetLease(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	leaseID := vars["id"]
	lease, err := DbSvc.GetLeaseByID(leaseID)
	if err != nil {
		log.Printf("Error Getting Lease for Id: %s", leaseID)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed Get on Lease: %s", leaseID))
		// http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lease == nil {
		log.Printf("Error Getting Lease for Id: %s", err)
		http.Error(w, fmt.Sprintf("Error Getting Lease by ID: %s", leaseID), http.StatusNotFound)
		return
	}

	leaseResponse := response.LeaseResponse(*lease)
	json.NewEncoder(w).Encode(leaseResponse)
}

func ServerErrorWithResponse(w http.ResponseWriter, message string) {
	CreateAPIErrorResponse(
		w,
		500,
		CreateErrorResponse("ServerError", message),
	)
}

// CreateErrorResponse creates and returns a formatted JSON string of the
// structured ErrorResponse
func CreateErrorResponse(code string, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorBase{
			Code:    code,
			Message: message,
		},
	}
}

type ErrorResponse struct {
	Error ErrorBase `json:"error"`
}

type ErrorBase struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func CreateAPIResponse(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// createAPIErrorResponse is a helper function to create and return a valid error
// response message for the API
func CreateAPIErrorResponse(w http.ResponseWriter, responseCode int,
	errResp ErrorResponse) {
	// Create the Error Response
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		CreateAPIResponse(w, http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Return an error
	CreateAPIResponse(w, responseCode, string(apiResponse))
}
