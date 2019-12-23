package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/dceerr"
)

// WriteAPIErrorResponse writes an error to the ResponseWriter
func WriteAPIErrorResponse(w http.ResponseWriter, err error) {
	if debug {
		log.Printf("%+v", err)
	} else {
		log.Printf("%v", err)
	}

	switch t := err.(type) {
	case dceerr.APICode:
		WriteAPIResponse(w, t.APICode(), err)
		return
	}
	WriteAPIResponse(
		w,
		http.StatusInternalServerError,
		dceerr.NewInternalServer("unknown error", err),
	)
}

// WriteAPIResponse writes the response out to the provided ResponseWriter
func WriteAPIResponse(w http.ResponseWriter, status int, body interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
