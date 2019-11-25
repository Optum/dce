package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

func GetAuthPage(w http.ResponseWriter, r *http.Request) {
	lp := filepath.Join("views", "index.html")

	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to load web page: %s", err)
		log.Print(errorMessage)
		WriteServerErrorWithResponse(w, errorMessage)
	}

	env := struct {
		SITE_PATH_PREFIX         string
		APIGW_DEPLOYMENT_NAME    string
		IDENTITY_POOL_ID         string
		USER_POOL_PROVIDER_NAME  string
		USER_POOL_CLIENT_ID      string
		USER_POOL_APP_WEB_DOMAIN string
		USER_POOL_ID             string
		AWS_CURRENT_REGION       string
	}{
		SITE_PATH_PREFIX:         sitePathPrefix,
		APIGW_DEPLOYMENT_NAME:    apigwDeploymentName,
		IDENTITY_POOL_ID:         identityPoolID,
		USER_POOL_PROVIDER_NAME:  userPoolProviderName,
		USER_POOL_CLIENT_ID:      userPoolClientID,
		USER_POOL_APP_WEB_DOMAIN: userPoolAppWebDomain,
		USER_POOL_ID:             userPoolID,
		AWS_CURRENT_REGION:       awsCurrentRegion,
	}
	if err := tmpl.Execute(w, env); err != nil {
		errorMessage := fmt.Sprintf("Failed to load web page: %s", err)
		log.Print(errorMessage)
		WriteServerErrorWithResponse(w, errorMessage)
	}
}

func GetAuthPageAssets(w http.ResponseWriter, r *http.Request) {
	fs := http.FileServer(http.Dir("./public"))
	sp := http.StripPrefix("/auth/public", fs)
	sp.ServeHTTP(w, r)
}
