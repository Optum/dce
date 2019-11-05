package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// EmptyQueryString - Empty query string to prevent
var EmptyQueryString []string

// Route - A route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	Queries     []string
	HandlerFunc http.HandlerFunc
}

// Routes - The list of Routes
type Routes []Route

// NewRouter - Create a new router
func NewRouter(routes Routes) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)

		for i := 0; i < len(route.Queries); i++ {
			router.Queries(route.Queries[i], fmt.Sprintf("{%s}", route.Queries[i]))
		}
	}

	router.Use(handlers.CORS())
	router.Use(commonHeaders)
	return router
}

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}
