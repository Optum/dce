package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
)

// WriteServerErrorWithResponse - Writes a server error with the specific message.
func WriteServerErrorWithResponse(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusInternalServerError,
		"ServerError",
		message,
	)
}

// WriteAPIErrorResponse - Writes the error response out to the provided ResponseWriter
func WriteAPIErrorResponse(w http.ResponseWriter, responseCode int,
	errCode string, errMessage string) {
	// Create the Error Response
	errResp := response.CreateErrorResponse(errCode, errMessage)
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		WriteAPIResponseEOL(w, http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Write an error
	WriteAPIResponseEOL(w, responseCode, string(apiResponse))
}

// WriteAPIResponseEOL - Writes the response out to the provided ResponseWriter
func WriteAPIResponseEOL(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// WriteAlreadyExistsError - Writes the already exists error.
func WriteAlreadyExistsError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusConflict,
		"AlreadyExistsError",
		"The requested resource cannot be created, as it conflicts with an existing resource",
	)
}

// WriteRequestValidationError - Writes a request validate error with the given message.
func WriteRequestValidationError(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusBadRequest,
		"RequestValidationError",
		message,
	)
}

// WriteNotFoundError - Writes a request validate error with the given message.
func WriteNotFoundError(w http.ResponseWriter) {
	WriteAPIErrorResponse(
		w,
		http.StatusNotFound,
		"NotFound",
		"The requested resource could not be found.",
	)
}
