package main

import (
	"fmt"
	"github.com/Optum/dce/pkg/api/response"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func GetAuthPage(w http.ResponseWriter, r *http.Request) {
	lp := filepath.Join("views", "index.html")

	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to load web page: %s", err)
		log.Print(errorMessage)
		response.WriteServerErrorWithResponse(w, errorMessage)
	}
	if err := tmpl.Execute(w, Config); err != nil {
		errorMessage := fmt.Sprintf("Failed to load web page: %s", err)
		log.Print(errorMessage)
		response.WriteServerErrorWithResponse(w, errorMessage)
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
}

func GetAuthPageAssets(w http.ResponseWriter, r *http.Request) {
	fs := http.FileServer(http.Dir("./public"))
	sp := http.StripPrefix("/auth/public", fs)

	splitStr := strings.Split(r.URL.Path, ".")
	ext := splitStr[len(splitStr)-1]
	var contentType string
	switch ext {
	case "css":
		contentType = "text/css"
	case "js":
		contentType = "text/javascript"
	default:
		contentType = "application/json"
	}

	w.Header().Set("Content-Type", contentType)
	sp.ServeHTTP(w, r)
}
