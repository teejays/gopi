package gopi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/teejays/goku-util/log"
)

// Route represents a standard route object
type Route struct {
	Method       string
	Version      int
	Path         string
	HandlerFunc  http.HandlerFunc
	Authenticate bool
}

// StartServer initializes and runs the HTTP server
func StartServer(ctx context.Context, addr string, port int, routes []Route, authMiddlewareFunc MiddlewareFunc, preMiddlewareFuncs, postMiddlewareFuncs []MiddlewareFunc) error {

	m, err := GetHandler(ctx, routes, authMiddlewareFunc, preMiddlewareFuncs, postMiddlewareFuncs)
	if err != nil {
		return fmt.Errorf("could not setup the http handler: %c", err)
	}

	http.Handle("/", m)

	// Start the server
	log.Info(ctx, "[Gopi] HTTP Server listening", "address", addr, "port", port)

	err = http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), nil)
	if err != nil {
		return fmt.Errorf("HTTP Server failed to start or continue running: %v", err)
	}

	return nil

}

// GetHandler constructs a HTTP handler with all the routes and middleware funcs configured
func GetHandler(ctx context.Context, routes []Route, authMiddlewareFunc MiddlewareFunc, preMiddlewareFuncs, postMiddlewareFuncs []MiddlewareFunc) (http.Handler, error) {

	// Initiate a router
	m := mux.NewRouter().PathPrefix("api").Subrouter()

	// Enable CORS
	// TODO: Have tighter control over CORS policy, but okay for
	// as long as we're just developing. This shouldn't really go on prod.
	originsOk := handlers.AllowedOrigins([]string{"*"})
	credsOk := handlers.AllowCredentials()
	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "authorization"})
	methodsOk := handlers.AllowedMethods([]string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch})
	corsEnabler := handlers.CORS(originsOk, credsOk, headersOk, methodsOk)

	// Register routes to the handler
	// Set up pre handler middlewares
	for _, mw := range preMiddlewareFuncs {
		m.Use(mux.MiddlewareFunc(mw))
	}

	// Create an authenticated subrouter
	var a *mux.Router
	if authMiddlewareFunc != nil {
		a = m.PathPrefix("").Subrouter()
		a.Use(mux.MiddlewareFunc(authMiddlewareFunc))
	}

	// Range over routes and register them
	for _, route := range routes {
		// If the route is supposed to be authenticated, use auth mux
		r := m
		if route.Authenticate {
			if a == nil {
				// We marked a route as requiring authentication but provided no auth middleware func :(
				return nil, fmt.Errorf("route for %s has authentication flag set but no authentication middleware has been provided", route.Path)
			}
			r = a
		}
		// Register the route
		log.Info(ctx, "[Gopi] Registering endpoint", "path", GetRoutePattern(route), "method", route.Method)

		r.HandleFunc(GetRoutePattern(route), route.HandlerFunc).
			Methods(route.Method)
	}

	// Set up pre handler middlewares
	for _, mw := range postMiddlewareFuncs {
		m.Use(mux.MiddlewareFunc(mw))
	}

	mc := corsEnabler(m)

	return mc, nil
}

// MiddlewareFunc can be inserted in a server for processing
type MiddlewareFunc mux.MiddlewareFunc

// LoggerMiddleware is a http.Handler middleware function that logs any request received
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		log.DebugNoCtx("[Gopi] HTTP request received", "http_method", r.Method, "path", r.URL.Path)
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// SetJSONHeaderMiddleware sets the header for the response
func SetJSONHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the header
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// returns the url match pattern for the route
func GetRoutePattern(r Route) string {
	return fmt.Sprintf("/v%d/%s", r.Version, r.Path)
}
