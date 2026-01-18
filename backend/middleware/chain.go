// ABOUTME: Middleware chaining utility for composing HTTP middleware
// ABOUTME: Applies middleware in declaration order (first is outermost)

package middleware

import "net/http"

// Chain applies middleware functions to a handler in order.
// The first middleware in the list is the outermost (executes first).
// Example: Chain(handler, logging, cors) applies as: logging(cors(handler))
func Chain(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
