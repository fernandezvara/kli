// cli/middleware.go
package kli

import (
	"fmt"
	"log"
	"time"
)

// LoggingMiddleware creates middleware that logs command execution with timing
// The logger function receives the command context and execution duration
func LoggingMiddleware(logger func(*CommandContext, time.Duration)) CommandMiddleware {
	return func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			start := time.Now()

			// Log command start
			log.Printf("Starting command: %s", ctx.Command)
			if ctx.SubCommand != "" {
				log.Printf("Subcommand: %s", ctx.SubCommand)
			}

			// Execute next in chain
			err := next(ctx)

			// Log completion with timing
			duration := time.Since(start)
			logger(ctx, duration)

			return err
		}
	}
}

// AuthMiddleware creates middleware that validates authentication before command execution
// The auth function should return nil if authentication succeeds, or an error if it fails
func AuthMiddleware(authFunc func(*CommandContext) error) CommandMiddleware {
	return func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			// Check authentication before executing command
			if err := authFunc(ctx); err != nil {
				log.Printf("Authentication failed for command %s: %v", ctx.Command, err)
				return fmt.Errorf("authentication failed: %w", err)
			}

			log.Printf("Authentication successful for command %s", ctx.Command)

			// Auth passed, execute command
			return next(ctx)
		}
	}
}

// TimingMiddleware creates middleware that measures and stores execution timing
// The timing is stored in ctx for other middleware to use
func TimingMiddleware() CommandMiddleware {
	return func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			start := time.Now()
			err := next(ctx)
			duration := time.Since(start)

			// Store timing in context for other middleware
			ctx.Set("duration", duration)

			log.Printf("Command %s took %v", ctx.Command, duration)

			return err
		}
	}
}

// ConditionalMiddleware creates middleware that only applies when the condition is true
// This allows for sophisticated conditional middleware application
func ConditionalMiddleware(condition func(*CommandContext) bool, middleware CommandMiddleware) CommandMiddleware {
	return func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			if condition(ctx) {
				return middleware(next)(ctx)
			}
			return next(ctx)
		}
	}
}

// RecoveryMiddleware creates middleware that recovers from panics
// This prevents the entire application from crashing due to panics in commands
func RecoveryMiddleware() CommandMiddleware {
	return func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic recovered in command %s: %v", ctx.Command, r)

					// Store panic in context for error handling middleware
					ctx.Set("panic", r)
				}
			}()

			return next(ctx)
		}
	}
}

// MetricsMiddleware creates middleware for collecting command metrics
// This is useful for monitoring and analytics
func MetricsMiddleware(metricsCollector func(*CommandContext, time.Duration, error)) CommandMiddleware {
	return func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			start := time.Now()
			err := next(ctx)
			duration := time.Since(start)

			// Collect metrics
			metricsCollector(ctx, duration, err)

			return err
		}
	}
}
