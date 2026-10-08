// cli/middleware_chain_test.go
package kli

import (
	"errors"
	"testing"
)

func TestMiddlewareChain_ApplyCommandOnly(t *testing.T) {
	chain := newMiddlewareChain()

	// Create a command with middleware
	cmd := &Command{
		Name: "test",
		Func: func(ctx *CommandContext) error {
			ctx.Set("executed", true)
			return nil
		},
		Middleware: []CommandMiddleware{
			// Middleware that adds a value to context
			func(next CommandFunc) CommandFunc {
				return func(ctx *CommandContext) error {
					ctx.Set("middleware1", "value1")
					return next(ctx)
				}
			},
			// Middleware that adds another value
			func(next CommandFunc) CommandFunc {
				return func(ctx *CommandContext) error {
					ctx.Set("middleware2", "value2")
					return next(ctx)
				}
			},
		},
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply middleware only
	finalFunc := chain.ApplyCommandOnly(cmd, cmd.Func)

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("ApplyCommandOnly() returned error: %v", err)
	}

	// Check that middleware was applied in correct order
	if val, exists := ctx.GetData("middleware1"); !exists || val != "value1" {
		t.Error("Middleware 1 was not applied correctly")
	}

	if val, exists := ctx.GetData("middleware2"); !exists || val != "value2" {
		t.Error("Middleware 2 was not applied correctly")
	}

	if val, exists := ctx.GetData("executed"); !exists || val != true {
		t.Error("Command function was not executed")
	}
}

func TestMiddlewareChain_ApplyGlobalOnly(t *testing.T) {
	chain := newMiddlewareChain()

	// Create global middleware
	globalMiddleware := []CommandMiddleware{
		func(next CommandFunc) CommandFunc {
			return func(ctx *CommandContext) error {
				ctx.Set("global1", "global_value1")
				return next(ctx)
			}
		},
		func(next CommandFunc) CommandFunc {
			return func(ctx *CommandContext) error {
				ctx.Set("global2", "global_value2")
				return next(ctx)
			}
		},
	}

	// Create a base function
	baseFunc := func(ctx *CommandContext) error {
		ctx.Set("base_executed", true)
		return nil
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply global middleware only
	finalFunc := chain.ApplyGlobalOnly(globalMiddleware, baseFunc)

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("ApplyGlobalOnly() returned error: %v", err)
	}

	// Check that global middleware was applied
	if val, exists := ctx.GetData("global1"); !exists || val != "global_value1" {
		t.Error("Global middleware 1 was not applied correctly")
	}

	if val, exists := ctx.GetData("global2"); !exists || val != "global_value2" {
		t.Error("Global middleware 2 was not applied correctly")
	}

	if val, exists := ctx.GetData("base_executed"); !exists || val != true {
		t.Error("Base function was not executed")
	}
}

func TestMiddlewareChain_Apply_Combined(t *testing.T) {
	chain := newMiddlewareChain()

	// Create a command with middleware
	cmd := &Command{
		Name: "test",
		Func: func(ctx *CommandContext) error {
			ctx.Set("command_executed", true)
			return nil
		},
		Middleware: []CommandMiddleware{
			func(next CommandFunc) CommandFunc {
				return func(ctx *CommandContext) error {
					ctx.Set("cmd_middleware", "cmd_value")
					return next(ctx)
				}
			},
		},
	}

	// Create global middleware
	globalMiddleware := []CommandMiddleware{
		func(next CommandFunc) CommandFunc {
			return func(ctx *CommandContext) error {
				ctx.Set("global_middleware", "global_value")
				return next(ctx)
			}
		},
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply combined middleware: global wraps command-specific
	finalFunc := chain.ApplyGlobalOnly(globalMiddleware, chain.ApplyCommandOnly(cmd, cmd.Func))

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("middleware chain returned error: %v", err)
	}

	// Check that both global and command middleware were applied
	if val, exists := ctx.GetData("global_middleware"); !exists || val != "global_value" {
		t.Error("Global middleware was not applied correctly")
	}

	if val, exists := ctx.GetData("cmd_middleware"); !exists || val != "cmd_value" {
		t.Error("Command middleware was not applied correctly")
	}

	if val, exists := ctx.GetData("command_executed"); !exists || val != true {
		t.Error("Command function was not executed")
	}
}

func TestMiddlewareChain_EmptyMiddleware(t *testing.T) {
	chain := newMiddlewareChain()

	// Create command without middleware
	cmd := &Command{
		Name: "test",
		Func: func(ctx *CommandContext) error {
			ctx.Set("executed", true)
			return nil
		},
		Middleware: []CommandMiddleware{},
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply middleware (should be no-op)
	finalFunc := chain.ApplyCommandOnly(cmd, cmd.Func)

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("ApplyCommandOnly() with empty middleware returned error: %v", err)
	}

	if val, exists := ctx.GetData("executed"); !exists || val != true {
		t.Error("Command function was not executed")
	}
}

func TestMiddlewareChain_NilCommand(t *testing.T) {
	chain := newMiddlewareChain()

	baseFunc := func(ctx *CommandContext) error {
		ctx.Set("executed", true)
		return nil
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply with nil command (should return base function)
	finalFunc := chain.ApplyCommandOnly(nil, baseFunc)

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("ApplyCommandOnly() with nil command returned error: %v", err)
	}

	if val, exists := ctx.GetData("executed"); !exists || val != true {
		t.Error("Base function was not executed when command is nil")
	}
}

func TestMiddlewareChain_MiddlewareErrorHandling(t *testing.T) {
	chain := newMiddlewareChain()

	// Create middleware that returns an error
	errorMiddleware := func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			return errors.New("middleware error")
		}
	}

	cmd := &Command{
		Name: "test",
		Func: func(ctx *CommandContext) error {
			ctx.Set("executed", true)
			return nil
		},
		Middleware: []CommandMiddleware{errorMiddleware},
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply middleware
	finalFunc := chain.ApplyCommandOnly(cmd, cmd.Func)

	// Execute the function
	err := finalFunc(ctx)

	// Check that error was returned
	if err == nil {
		t.Error("Expected error from middleware, but got nil")
	}

	if err.Error() != "middleware error" {
		t.Errorf("Expected 'middleware error', got %v", err)
	}

	// Check that command function was not executed
	if _, exists := ctx.GetData("executed"); exists {
		t.Error("Command function should not have been executed when middleware returned error")
	}
}

func TestMiddlewareChain_MiddlewareOrder(t *testing.T) {
	chain := newMiddlewareChain()

	var executionOrder []string

	// Create middleware that tracks execution order
	middleware1 := func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			executionOrder = append(executionOrder, "middleware1")
			return next(ctx)
		}
	}

	middleware2 := func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			executionOrder = append(executionOrder, "middleware2")
			return next(ctx)
		}
	}

	cmd := &Command{
		Name: "test",
		Func: func(ctx *CommandContext) error {
			executionOrder = append(executionOrder, "command")
			return nil
		},
		Middleware: []CommandMiddleware{middleware1, middleware2},
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply middleware
	finalFunc := chain.ApplyCommandOnly(cmd, cmd.Func)

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("ApplyCommandOnly() returned error: %v", err)
	}

	// Check execution order: middleware1 -> middleware2 -> command
	// (because we apply in reverse order, so middleware2 wraps middleware1 wraps command)
	expectedOrder := []string{"middleware1", "middleware2", "command"}

	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d executions, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("Expected execution order %v, got %v", expectedOrder, executionOrder)
			break
		}
	}
}
