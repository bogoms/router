# router
Simple and fast router for Go (Golang) HTTP server.

## Using router
`router` implements `http.HandlerFunc`, so it can be used with `http.ListenAndServe()`.
Here is a simple example of `router` usage for REST API application:

```go
type appContext struct {
  // App specific data, like DB context.
  // Not needed for router, but can be useful in real project.
}

// Handler function of type router.HandlerFunc. router.Params will contain all parameters
// encoded in URI including named parameter if present.
func (c *appContext) testHandler(w http.ResponseWriter, r *http.Request, ps router.Params) {
	writeJson(w, map[string]string{"message": "API test"})
}

// Another handler function.
func (c *appContext) testIdHandler(w http.ResponseWriter, r *http.Request, ps router.Params) {
	writeJson(w, ps)
}

// Helper function that shows how to perform some steps before and after request handling
// (for example logging).
func newHandler(h router.HandlerFunc) router.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ps router.Params) {

		// TODO: do some steps before request handling.

		// Call original request handler.
		h(w, r, ps)

		// TODO: do some steps after request handling.
	}
}

// Panic handler function will be called ina case of panic during request handling.
// If this function is not implemented - then router will recover from panic and 
// set HTTP status code to 500 Internal Server Error.
func (c *appContext) panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	// Set HTTP status code to 500 Internal Server Error.
	w.WriteHeader(http.StatusInternalServerError)

	// TODO: Log error.
}

// Writes JSON to response writer.
func writeJson(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func main() {
	// Create router.
	router := router.New()
	router.PanicHandler = c.panicHandler

	// Add API endpoints.
	if err := router.Get("/api/test", newHandler(c.testHandler)); err != nil {
		log.Fatalln(err)
	}

	if err := router.Get("/api/test/:id", newHandler(c.testIdHandler)); err != nil {
		log.Fatalln(err)
	}

	// Start HTTP-server.
	err := http.ListenAndServe(":8000", router)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
```

## Adding routes
There is a main function to add routes:
```go
err := router.Handle("METHOD", "/path", routerHandlerFunc)
```

For the sake of convenience, there are shortcuts for the most used HTTP methods:
```go
// Equivalent to router.Handle("GET", "/path", getHandlerFunc)
err = router.Get("/path", getHandlerFunc)       

// Equivalent to router.Handle("PUT", "/path", putHandlerFunc)
err = router.Put("/path", putHandlerFunc)

// Equivalent to router.Handle("POST, "/path", postHandlerFunc)
err = router.Post("/path", postHandlerFunc)

// Equivalent to router.Handle("DELETE, "/path", deleteHandlerFunc)
err = router.Delete("/path", deleteHandlerFunc)
```
