package httpserve

import (
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

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

// newRouterHandler will return a new httprouter.Handle
func newRouterHandler(hs []Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Create context
		ctx := newContext(w, r, p)
		// Get response from context by passing provided handlers
		resp := ctx.getResponse(hs)
		if ctx.wasAdopted(resp) {
			return
		}
		defer r.Body.Close()

		// Respond using context
		ctx.respond(resp)

		statusCode := 200
		if resp != nil {
			statusCode = resp.StatusCode()
		}
		// Process context hooks
		ctx.processHooks(statusCode)
	}
}

// newHTTPHandler will return a new http.Handler
func newHTTPHandler(hs []Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context
		ctx := newContext(w, r, httprouter.Params{})
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
