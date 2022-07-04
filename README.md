# GOPI

Go makes starting HTTP servers and spawning API endpoints super straightforward. For a simple servers, it's totally easy peasy. For bigger projects, however, which may require a server with many endpoints e.g. REST endpoints for 10-20 entities, it quickly becomes painful to manage existing and bring up new endpoints. At least,that's the pain I went through.

GOPI is a library to make that pain go away, and make HTTP endpoints management super manageable. There are a few simple concepts:

### Route
A Route is an entire specification required to setup a particular endpoint. It basically needs four things:
 
 - Method (string): GET, POST, PUT, DELETE etc.
 - Path (string): The endpoint for this particular API
 - Version (int): Allows us to version the particular route by pre-pending "v1" or "v2" etc. to the path.
 - HandlerFunc (http.HandlerFunc): The Main handler function for this route
 - Authenticate (bool): Optional - if passed as true, the optional middleware setup for authentication will be called.

 e.g. 
 ```golang
    routes := []gopi.Route{
        {
            Method:      http.MethodGet,
            Version:     1,
            Path:        "ping",
            HandlerFunc: HandlePingRequest,
        }
    }
    
    func HandlePingRequest(w http.ResponseWriter, r *http.Request) {
	    w.Write([]byte(`Pong!`))
    }
```

### Middleware Func
Middleware Funcs are pieces of code that are run right before/after a request for a particular route is being processed by its handler. There are three kinds of Middleware funcs:
 1. Pre Middleware Funcs which are run before the HTTP request is passed to the handler
 2. Post Middleware Funcs, which are run after the request has been returned from handler
 3. Authenticate Middleware, a special kind of Pre Middleware which is run only when 'Authenticate' is set to true. 

GOPI comes with some standard useful Middleware Funcs that are helpful in setting up a REST server e.g. `api.LoggerMiddleware` (which logs all the requests to Std. Out), `api.SetJSONHeaderMiddleware` (which sets the `Content-Type: application/json` header for the response). No standard authenticate middleware is provided with the library yet, so users are free to implement their own. 

### Server
A server takes in a bunch of routes and optionally some middleware funcs, and sets up a HTTP server for them.

```golang
// params: address, port, authenticate middleware, pre-middlewares, post-middlewares
err := gopi.StartServer("127.0.0.1", 8080, routes, nil, nil, nil)
if err != nil {
    log.Fatal(err)
}
```

## Example


```golang
package main

import (
    "log"
	"net/http"

	api "github.com/teejays/gopi"

)

func main() {

    // Define Routes
    routes := []api.Route{
        
        // Ping Handler (/v1/ping)
        {
            Method:      http.MethodGet, // Get or Post, or what..
            Version:     1,
            Path:        "ping", // endpoint
            HandlerFunc: HandlePingRequest, // Need to define these handlers
        },
        // Ping Handler - Authenticated (v2/ping)
        {
            Method:       http.MethodGet,
            Version:      2,
            Path:         "ping",
            HandlerFunc:  HandlePingRequest,
            Authenticate: true, // if you want this endpoint to always run the 'Authenticate' middleware
        },
        // Create Something
        {
            Method:       http.MethodPost,
            Version:      1,
            Path:         "something",
            HandlerFunc:  HandleCreateSomething,
        },
        // Get Something by ID
        {
            Method:       http.MethodGet,
            Version:      1,
            Path:         "something/{id}",
            HandlerFunc:  HandleGetSomething,
        },
    }

	// - Middlewares (defined above outside this function)
    preMiddlewareFuncs := []api.MiddlewareFunc{api.MiddlewareFunc(api.LoggerMiddleware)}
    postMiddlewareFuncs := []api.MiddlewareFunc{api.SetJSONHeaderMiddleware}
    authMiddlewareFunc := nil

    err := api.StartServer("127.0.0.1", 8080, routes, authMiddlewareFunc, preMiddlewareFuncs, postMiddlewareFuncs)
    if err != nil {
        log.Fatal(err)
    }

    return

}

// Define your handlers: the requests for the routes are eventually passed to the handlers specified in the route

func HandlePingRequest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`Pong!`))
}

func HandleCreateSomething(w http.ResponseWriter, r *http.Request) {
    // Do stuff...

    // for now we just...
	api.WriteResponse(w, http.StatusCreated, "something is created")
}

func HandleGetSomething(w http.ResponseWriter, r *http.Request) {
    // Do stuff...

    // for now we just...
	api.WriteError(w, http.StatusNotFound, err, false, nil)
}

```


