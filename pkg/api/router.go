package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/common"
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

var (
	debug bool
)

func init() {
	config := common.DefaultEnvConfig{}
	debug = config.GetEnvBoolVar("DEBUG", false)
}

// Routes - The list of Routes
type Routes []Route

// NewRouter - Create a new router
func NewRouter(routes Routes) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc

		if debug {
			log.Printf("Adding route %s with pattern: \"%s\"", route.Name, route.Pattern)
		}

		if len(route.Queries) == 0 {

			router.
				Methods(route.Method).
				Path(route.Pattern).
				Name(route.Name).
				Handler(handler)
		} else {
			queryStringPairs := make([]string, len(route.Queries)*2)
			for i := 0; i < len(route.Queries); i++ {
				queryStringPairs[i] = route.Queries[i]
				queryStringPairs[i+1] = fmt.Sprintf("{%s}", route.Queries[i])
			}

			router.
				Methods(route.Method).
				Path(route.Pattern).
				Name(route.Name).
				Queries(queryStringPairs...).
				Handler(handler)
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
