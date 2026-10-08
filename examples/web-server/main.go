// Web Server Example - Complete Production Application
// Demonstrates: config-only mode, validation, secrets, file loading, source priority, builder patterns
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fernandezvara/kli"
)

func main() {
	cfg := kli.New()

	// Server configuration with comprehensive validation
	cfg.Define("PORT").
		Int64().
		Env("PORT").
		Flag("port").
		File("port_in_file").
		Default(int64(8080)).
		Range(1, 65535).
		Description("HTTP server port")

	cfg.Define("HOST").
		String().
		Env("HOST").
		Flag("host").
		File("host_in_file").
		Default("localhost").
		Description("Server host")

	cfg.Define("BASE_URL").
		String().
		Env("BASE_URL").
		Flag("base-url").
		Required().
		Regexp(`^https?://`).
		Description("Public base URL of the service")

	cfg.Define("DATABASE_URL").
		String().
		Env("DATABASE_URL").
		Required().
		Secret().
		MinLength(10).
		Description("Database connection string")

	cfg.Define("REDIS_URL").
		String().
		Env("REDIS_URL").
		Secret().
		Description("Redis connection URL")

	cfg.Define("LOG_LEVEL").
		String().
		Env("LOG_LEVEL").
		Flag("log-level").
		Default("info").
		OneOf("debug", "info", "warn", "error").
		Description("Logging level")

	cfg.Define("ACCESS_TOKEN_TTL").
		Duration().
		Env("ACCESS_TOKEN_TTL").
		Default(15 * time.Minute).
		MinDuration(1 * time.Minute).
		MaxDuration(24 * time.Hour).
		Description("Access token lifetime")

	cfg.Define("CORS_ORIGINS").
		StringSlice().
		Env("CORS_ORIGINS").
		Flag("cors-origins").
		Default([]string{"http://localhost:3000"}).
		Delimiter(",").
		Description("Allowed CORS origins")

	cfg.Define("JWT_SIGNING_KEY").
		String().
		Env("JWT_SIGNING_KEY").
		Required().
		Secret().
		MinLength(32).
		Description("Secret key for signing access tokens")

	cfg.Define("ENVIRONMENT").
		String().
		Env("ENVIRONMENT").
		Flag("env").
		Default("development").
		OneOf("development", "staging", "production").
		Description("Application environment")

	cfg.Define("MAX_CONNECTIONS").
		Int64().
		Env("MAX_CONNECTIONS").
		Default(int64(100)).
		Range(1, 1000).
		Description("Maximum database connections")

	cfg.Define("ENABLE_METRICS").
		Bool().
		Env("ENABLE_METRICS").
		Flag("enable-metrics").
		Default(true).
		Description("Enable metrics collection")

	cfg.Define("LOG_PERMS").
		String().
		Env("LOG_PERMS").
		Default("0640").
		Regexp(`^0[0-7]{3}$`).
		Description("Log file assigned permissions (octal string)")

	// Add empty string command for config-only mode
	cfg.Command("").
		Func(func(ctx *kli.CommandContext) error {
			fmt.Printf("Web Server Starting!\n")

			// Get basic configuration with fallbacks
			if port, err := kli.Get[int64](ctx, "PORT"); err == nil {
				fmt.Printf("   Port: %d\n", port)
			} else {
				fmt.Printf("   Port: 8080 (default)\n")
			}

			if host, err := kli.Get[string](ctx, "HOST"); err == nil {
				fmt.Printf("   Host: %s\n", host)
			} else {
				fmt.Printf("   Host: localhost (default)\n")
			}

			if logLevel, err := kli.Get[string](ctx, "LOG_LEVEL"); err == nil {
				fmt.Printf("   Log Level: %s\n", logLevel)
			} else {
				fmt.Printf("   Log Level: info (default)\n")
			}

			// Check for secrets
			if dbSecret := ctx.GlobalConfig.GetSecret("DATABASE_URL"); dbSecret.IsSet() {
				fmt.Printf("   Database: %s\n", maskSecret(dbSecret.String()))
			} else {
				fmt.Printf("   Database: not configured\n")
			}

			if jwtSecret := ctx.GlobalConfig.GetSecret("JWT_SIGNING_KEY"); jwtSecret.IsSet() {
				fmt.Printf("   JWT Key: configured (%d bytes)\n", jwtSecret.Size())
			} else {
				fmt.Printf("   JWT Key: not configured\n")
			}

			baseUrl, err := kli.Get[string](ctx, "BASE_URL")
			if err == nil {
				fmt.Printf("   Base URL: %s\n", baseUrl)
			} else {
				fmt.Printf("   Base URL: not configured\n")
			}

			fmt.Printf("\nConfiguration loaded successfully!\n")
			fmt.Printf("Web server ready to start!\n")

			return nil
		}).
		ShortHelp("Start the web server").
		LongHelp(`Starts the web server with the specified configuration.

This is a  web server that supports:
- HTTP/HTTPS with configurable timeouts  
- Database connection pooling
- Metrics collection
- Environment variable configuration
- File-based configuration
- Secret management

Use --help or --full-help to see all available options.`).
		Config(func(cc *kli.CommandConfig) {
			// Add basic configuration for demo
			cc.Define("PORT").
				Int64().
				Env("PORT").
				Flag("port").
				Default(int64(8080)).
				Description("HTTP server port")

			cc.Define("HOST").
				String().
				Env("HOST").
				Flag("host").
				Default("localhost").
				Description("Server host")

			cc.Define("LOG_LEVEL").
				String().
				Env("LOG_LEVEL").
				Flag("log-level").
				Default("info").
				OneOf("debug", "info", "warn", "error").
				Description("Logging level")

			cc.Define("DATABASE_URL").
				String().
				Env("DATABASE_URL").
				Secret().
				Description("Database connection URL")

			cc.Define("JWT_SIGNING_KEY").
				String().
				Env("JWT_SIGNING_KEY").
				Secret().
				Description("Secret key for signing access tokens")
		})

	// Execute configuration (single unified entry point)
	if err := cfg.Execute(os.Args); err != nil {
		os.Exit(1)
	}
	defer cfg.Destroy()
}

// Helper function to mask secrets in output
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}
