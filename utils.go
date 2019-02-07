package httpserve

import (
	"fmt"
	"io"
	"net/http"
)

// newHandler will return a new Handler
func newHandler(hs []Handler) Handler {
	return func(ctx *Context) Response {
		// Get response from context by passing provided handlers
		return ctx.getResponse(hs)
	}
}

// Handler is the HTTP handler type
type Handler func(ctx *Context) Response

// Response is a response interface
type Response interface {
	StatusCode() (code int)
	ContentType() (contentType string)
	WriteTo(w io.Writer) (n int64, err error)
}

// Storage is used as a basic form of KV storage for a Context
// TODO: Determine with team if it seems valuable to change this to map[string]interface{}.
// I'd prefer if we can keep it as-is, due to the fact that map[string]string has much less
// GC overhead. Additionally, avoiding type assertion would be fantastic.
type Storage map[string]string

// Hook is a function called after the response has been completed to the requester
type Hook func(statusCode int, storage Storage)

// newHTTPHandler will return a new http.Handler
func newHTTPHandler(hs []Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context
		ctx := newContext(w, r, Params{})
		// Get response from context by passing provided handlers
		resp := ctx.getResponse(hs)
		if ctx.wasAdopted(resp) {
			return
		}
		defer r.Body.Close()
		// Respond using context
		ctx.respond(resp)
		// Process context hooks
		ctx.processHooks(resp.StatusCode())
	}
}

func newHTTPServer(h http.Handler, port uint16, c Config) *http.Server {
	var srv http.Server
	srv.Handler = h
	srv.Addr = fmt.Sprintf(":%d", port)
	srv.ReadTimeout = c.ReadTimeout
	srv.WriteTimeout = c.WriteTimeout
	srv.MaxHeaderBytes = c.MaxHeaderBytes
	return &srv
}

// getParts is used to split URLs into parts
func getParts(url string) (parts []string) {
	var (
		lastIndex int
		lastSlash int
	)

	fmt.Println("Getting parts", url)

	parts = make([]string, 0, 3)

	for i := 0; i < len(url); i++ {
		b := url[i]
		switch b {
		case ':':
			if lastSlash != i-1 {
				panic("parameters can only directly follow a forward slash")
			}

			part := url[lastIndex : i-1]
			parts = append(parts, part)
			lastIndex = i
		case '/':
			lastSlash = i
		}
	}

	fmt.Println("Before check", parts, lastIndex)
	parts = append(parts, url[lastIndex:])
	fmt.Println("Parts", parts)
	return
}

func notFoundHandler(ctx *Context) Response {
	return NewTextResponse(404, []byte("404, not found"))
}
