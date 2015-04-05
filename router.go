// Package router contains a simple and fast HTTP requests router that supports
// named parameters and specific handlers for different methods.
package router

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	wrongParamNameChars string = `/:`
)

// Router errors.
var (
	ErrParameterName    error = errors.New(fmt.Sprintf("router: parameter name cannot contain any of these charactars: %#q", wrongParamNameChars))
	ErrDuplicateHandler error = errors.New("router: handler for this path and method combination was already registered")
)

// A HandlerFunc represents an HTTP request handler function.
type HandlerFunc func(w http.ResponseWriter, r *http.Request, ps Params)

// A PanicHandlerFunc represents a special handler that will be
// called in case of panic during the request handlind.
type PanicHandlerFunc func(w http.ResponseWriter, r *http.Request, err interface{})

// A Params stores parameters that were passed as a part of URI.
type Params map[string][]string

// A Router stores all routes with corresponding API handler functions.
type Router struct {
	routes       map[string]*pathData
	PanicHandler PanicHandlerFunc
}

type pathMethods map[string]HandlerFunc

type pathData struct {
	path    string
	param   string
	methods pathMethods
}

// New initializes and returns a new router.
func New() *Router {
	return &Router{routes: map[string]*pathData{}}
}

// Get returns value for parameter with specified name.
// If parameter has several values, first one is returned.
func (ps Params) Get(name string) (string, bool) {
	v, ok := ps[name]
	if !ok || len(v) == 0 {
		return "", false
	}

	return v[0], true
}

// ServeHTTP handles the API request. It may perform some actions before
// and/or after calling the handler function.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Recover from panic.
	defer func() {
		if err := recover(); err != nil {
			// Check if custom panic handler present.
			if router.PanicHandler != nil {
				// Call the custom panic handler.
				router.PanicHandler(w, r, err)
			} else {
				// Write HTTP status code 500 Internal Server Error.
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}()

	// Handle HTTP request.
	router.doServeHTTP(w, r)
}

func (router *Router) doServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to get path data.
	pd, param := router.getPathData(r.URL.Path)
	if pd == nil {
		// Set status code to 404 Not Found.
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Try to get handler function for requested method.
	f, ok := pd.methods[r.Method]
	if !ok {
		// Create a list of allowed methods.
		allow := ""
		for m, _ := range pd.methods {
			allow += m
		}

		// Set Allow header.
		w.Header().Set("Allow", strings.TrimSuffix(allow, ", "))

		// Set status code to 405 Method Not Allowed.
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	// Parse form data.
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	// Get form parameters.
	params := Params(r.Form)

	// Add parameter sent as part of the URI if needed.
	if pd.param != "" {
		// Create new slice of values for parameter.
		s := []string{param}

		// Check if parameter name is used by form parameters.
		if v, ok := params[pd.param]; ok {
			// Insert the parameter sent as part of the URI at the beginning.
			// This is needed so that Params.Get() will return it.
			params[pd.param] = append(s, v...)
		} else {
			// Add new parameter name.
			params[pd.param] = s
		}
	}

	// Call the request handler.
	f(w, r, params)
}

// Handle sets an HTTP request handler for specific method and pattern.
// Patterns support named parameters, for example:
//
//		err := Handle("GET", "/api/users/:id", usersByIdHandler)
//
// will pass id parameter to handler. Only one named parameter is
// supported and it must be at the end of the URI.
//
func (r *Router) Handle(method string, pattern string, handler HandlerFunc) error {
	// Parse pattern.
	path, param, err := parsePattern(pattern)
	if err != nil {
		return err
	}

	// Try to get existing path data for the path.
	pd, ok := r.routes[path]
	if !ok {
		// Create new path data.
		pd = &pathData{
			path:    path,
			param:   param,
			methods: pathMethods{},
		}

		r.routes[path] = pd
	}

	// Check if handler for the path is already registred.
	if _, ok := pd.methods[method]; ok {
		return ErrDuplicateHandler
	}

	// Add handler for current method.
	pd.methods[method] = handler

	return nil
}

// Get adds handler for GET request.
func (r *Router) Get(pattern string, handler HandlerFunc) error {
	return r.Handle("GET", pattern, handler)
}

// Put adds handler for PUT request.
func (r *Router) Put(pattern string, handler HandlerFunc) error {
	return r.Handle("PUT", pattern, handler)
}

// Post adds handler for POST request.
func (r *Router) Post(pattern string, handler HandlerFunc) error {
	return r.Handle("POST", pattern, handler)
}

// Delete adds handler for DELETE request.
func (r *Router) Delete(pattern string, handler HandlerFunc) error {
	return r.Handle("DELETE", pattern, handler)
}

func normalizePath(p string) string {
	// Return root path if empty string is received.
	if len(p) == 0 {
		return "/"
	}

	// Trim slashes at the end.
	s := strings.TrimRight(p, "/")

	// Replace backslashes with slashes (\ -> /).
	s = strings.Replace(s, "\\", "/", -1)

	// Remove duplicate slashes (// -> /).
	for strings.Contains(s, "//") {
		s = strings.Replace(s, "//", "/", -1)
	}

	// Convert the string to lower.
	s = strings.ToLower(s)

	// Add leading slash if needed.
	if p[0] != '/' {
		s = "/" + s
	}

	// Return normalized path.
	return s
}

func parsePattern(pattern string) (string, string, error) {
	// Normalize pattern.
	path := normalizePath(pattern)

	// Check if pattern contains parameter.
	var param string
	i := strings.Index(path, "/:")
	if i >= 0 {
		// Get parameter name.
		param = path[i+2:]

		// Check parameter name.
		if strings.ContainsAny(param, wrongParamNameChars) {
			return "", "", ErrParameterName
		}

		// Remove parameter from the path, but keep "/:" at the end.
		path = path[:i+2]
	}

	// Return path and named parameter name.
	return path, param, nil
}

func (router *Router) getPathData(path string) (*pathData, string) {
	// Normalize path.
	path = normalizePath(path)

	// Try to get route without named parameter.
	if pd, ok := router.routes[path]; ok {
		// Return path data.
		return pd, ""
	}

	// Try to get route with named parameter.
	if i := strings.LastIndex(path, "/"); i > 0 {
		// Path with named parameter: remove parameter value and add "/:".
		if pd, ok := router.routes[path[:i]+"/:"]; ok {
			// Get parameter value.
			p := path[i+1:]

			// Return path data and named parameter name.
			return pd, p
		}
	}

	// Path data was not found.
	return nil, ""
}
