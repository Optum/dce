package main

import "net/http"

// Route - Structure of a Route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes - The list of Routes
type Routes []Route

var routes = Routes{
	Route{
		"GetLease",
		"GET",
		"/leases/{id}",
		GetLease,
	},
	Route{
		"CreateLease",
		"POST",
		"/leases",
		CreateLease,
	},
}
