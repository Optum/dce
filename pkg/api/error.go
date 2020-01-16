package api

import (
	"encoding/json"
	"io"
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

type writer interface {
	Write(writer io.Writer) error
}

// WriteAPIResponse writes the response out to the provided ResponseWriter
func WriteAPIResponse(httpWriter http.ResponseWriter, status int, body interface{}) {
	httpWriter.WriteHeader(status)

	if m, ok := body.(writer); ok {
		err := m.Write(httpWriter)
		if err != nil {
			log.Printf("error encoding and writing message: %+v", body)
			httpWriter.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(httpWriter).Encode(errors.NewInternalServer("error writing response", err))
		}
	} else {
		log.Printf("error encoding and writing message. No Write function exists for: %+v", body)
		httpWriter.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(httpWriter).Encode(errors.NewInternalServer("error writing response", nil))
	}

}
