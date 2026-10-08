// cli/middleware_chain.go
package kli

import "slices"

// MiddlewareChain manages middleware application and composition
type MiddlewareChain interface {
	// ApplyCommandOnly applies only command-specific middleware (for use within Command.Execute)
	ApplyCommandOnly(cmd *Command, baseFunc CommandFunc) CommandFunc

	// ApplyGlobalOnly applies only global middleware (for use within Config.Execute)
	ApplyGlobalOnly(globalMiddleware []CommandMiddleware, baseFunc CommandFunc) CommandFunc
}

// middlewareChain implements MiddlewareChain interface
type middlewareChain struct{}

// newMiddlewareChain creates a new MiddlewareChain instance
func newMiddlewareChain() MiddlewareChain {
	return &middlewareChain{}
}

// ApplyCommandOnly applies only command-specific middleware (for use within Command.Execute)
func (mc *middlewareChain) ApplyCommandOnly(cmd *Command, baseFunc CommandFunc) CommandFunc {
	if cmd == nil {
		return baseFunc
	}

	// Start with the base function
	finalFunc := baseFunc

	// Apply command-specific middleware in reverse order
	for _, v := range slices.Backward(cmd.Middleware) {
		finalFunc = v(finalFunc)
	}

	return finalFunc
}

// ApplyGlobalOnly applies only global middleware (for use within Config.Execute)
func (mc *middlewareChain) ApplyGlobalOnly(globalMiddleware []CommandMiddleware, baseFunc CommandFunc) CommandFunc {
	// Start with the base function
	finalFunc := baseFunc

	// Apply global middleware in reverse order
	for _, g := range slices.Backward(globalMiddleware) {
		finalFunc = g(finalFunc)
	}

	return finalFunc
}
