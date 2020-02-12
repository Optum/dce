package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/errors"
)

// WriteAPIErrorResponse writes an error to the ResponseWriter
func WriteAPIErrorResponse(w http.ResponseWriter, err error) {
	if debug {
		log.Printf("%+v", err)
	} else {
		log.Printf("%v", err)
	}

	switch t := err.(type) {
	case errors.HTTPCode:
		WriteAPIResponse(w, t.HTTPCode(), err)
		return
	}
	WriteAPIResponse(
		w,
		http.StatusInternalServerError,
		errors.NewInternalServer("unknown error", err),
	)
}

// WriteAPIResponse writes the response out to the provided ResponseWriter
func WriteAPIResponse(w http.ResponseWriter, status int, body interface{}) {
	w.WriteHeader(status)
	if body != nil {
		err := json.NewEncoder(w).Encode(body)
		if err != nil {
			log.Printf("error encoding and writing message: %+v", body)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(errors.NewInternalServer("error writing response", err))
		}
	}
}
