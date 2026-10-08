// cli/services_test.go
package kli

import (
	"errors"
	"testing"
	"time"
)

func TestCommandExecutor_Integration(t *testing.T) {
	// Test that CommandExecutor works correctly with the service factory
	services := newCommandServices()
	executor := services.Executor

	// Create a comprehensive command
	cmd := &Command{
		Name: "integration",
		Func: func(ctx *CommandContext) error {
			// Check configuration
			port, err := Get[int64](ctx, "PORT")
			if err != nil {
				return err
			}
			_ = port // Use the variable to avoid unused error

			// Check middleware context
			if _, exists := ctx.GetData("middleware_ran"); !exists {
				return errors.New("middleware did not run")
			}

			return nil
		},
		Definitions: map[string]*Definition{
			"PORT": {
				key:          "PORT",
				valueType:    TypeInt64,
				flag:         "port",
				description:  "Server port",
				defaultValue: int64(3000),
			},
		},
		Middleware: []CommandMiddleware{
			func(next CommandFunc) CommandFunc {
				return func(ctx *CommandContext) error {
					ctx.Set("middleware_ran", true)
					return next(ctx)
				}
			},
		},
	}

	ctx := NewCommandContext([]string{"--port", "5000"}, New(), "integration", "")

	// Execute the command
	result := executor.Execute(cmd, ctx, services)

	// Check that execution succeeded
	if result.Error != nil {
		t.Errorf("Integrated Execute failed: %v", result.Error)
	}

	// Verify configuration was processed
	port, err := Get[int64](ctx, "PORT")
	if err != nil {
		t.Errorf("Failed to get PORT: %v", err)
	}
	if port != 5000 {
		t.Errorf("Expected PORT=5000, got %d", port)
	}
}

func TestConfigProcessor_Integration(t *testing.T) {
	// Test that ConfigProcessor works correctly with the service factory
	services := newCommandServices()
	processor := services.ConfigProcessor

	// Create a simple command
	cmd := &Command{
		Name: "test",
		Definitions: map[string]*Definition{
			"DEBUG": {
				key:          "DEBUG",
				valueType:    TypeBool,
				flag:         "debug",
				description:  "Enable debug mode",
				defaultValue: false,
			},
		},
	}

	ctx := NewCommandContext([]string{"--debug", "true"}, New(), "test", "")

	// Test ProcessCommandConfig
	result := processor.ProcessCommandConfig(cmd, ctx)
	if result.Error != nil {
		t.Errorf("Integrated ProcessCommandConfig failed: %v", result.Error)
	}

	// Test ValidateRequiredFlags
	result = processor.ValidateRequiredFlags(cmd, ctx)
	if result.Error != nil {
		t.Errorf("Integrated ValidateRequiredFlags failed: %v", result.Error)
	}
}

func TestMiddlewareChain_Integration(t *testing.T) {
	// Test that MiddlewareChain works correctly with the service factory
	services := newCommandServices()
	chain := services.MiddlewareChain

	// Create timing middleware for testing
	timingMiddleware := func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			start := time.Now()
			err := next(ctx)
			duration := time.Since(start)
			ctx.Set("duration", duration)
			return err
		}
	}

	cmd := &Command{
		Name: "test",
		Func: func(ctx *CommandContext) error {
			time.Sleep(1 * time.Millisecond) // Simulate work
			return nil
		},
		Middleware: []CommandMiddleware{timingMiddleware},
	}

	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Apply middleware
	finalFunc := chain.ApplyCommandOnly(cmd, cmd.Func)

	// Execute the function
	err := finalFunc(ctx)

	// Check that execution succeeded
	if err != nil {
		t.Errorf("Integrated ApplyCommandOnly failed: %v", err)
	}

	// Check that timing middleware worked
	durationVal, exists := ctx.GetData("duration")
	if !exists {
		t.Error("Timing middleware did not set duration")
	} else if duration, ok := durationVal.(time.Duration); !ok || duration < 1*time.Millisecond {
		t.Error("Timing middleware did not measure correct duration")
	}
}
