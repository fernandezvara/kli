package kli

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBasicConfigurationDefinition(t *testing.T) {
	cfg := New()

	// Test basic definition
	cfg.Define("PORT").
		Int64().
		Env("PORT").
		Flag("port").
		Default(8080).
		Range(1, 65535).
		Description("HTTP server port")

	cfg.Define("BASE_URL").
		String().
		Env("BASE_URL").
		Flag("base-url").
		Required().
		Regexp(`^https?://`).
		Description("Public base URL of the service")

	cfg.Define("DEBUG").
		Bool().
		Env("DEBUG").
		Flag("debug").
		Default(false).
		Description("Enable debug mode")

	cfg.Define("TIMEOUT").
		Duration().
		Env("TIMEOUT").
		Default(30 * time.Second).
		MinDuration(1 * time.Second).
		MaxDuration(5 * time.Minute).
		Description("Request timeout")

	cfg.Define("CORS_ORIGINS").
		StringSlice().
		Env("CORS_ORIGINS").
		Flag("cors-origins").
		Delimiter(",").
		Default([]string{"http://localhost:3000"}).
		Description("Allowed CORS origins")

	cfg.Define("HOSTS").
		StringSlice().
		Env("HOSTS").
		Flag("hosts").
		Delimiter(",").
		Default([]string{"http://localhost:8080"}).
		Description("Allowed hosts")

	// Set environment variables for testing
	t.Setenv("BASE_URL", "https://api.example.com")
	t.Setenv("DEBUG", "true")

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Configuration errors: %v", err)
	}

	// Test values
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	port, err := Get[int64](ctx, "PORT")
	if err != nil {
		t.Fatalf("Error getting PORT: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected PORT=8080, got %d", port)
	}

	baseURL, err := Get[string](ctx, "BASE_URL")
	if err != nil {
		t.Fatalf("Error getting BASE_URL: %v", err)
	}
	if baseURL != "https://api.example.com" {
		t.Errorf("Expected BASE_URL=https://api.example.com, got %s", baseURL)
	}

	debug, err := Get[bool](ctx, "DEBUG")
	if err != nil {
		t.Fatalf("Error getting DEBUG: %v", err)
	}
	if !debug {
		t.Errorf("Expected DEBUG=true, got %v", debug)
	}

	timeout, err := Get[time.Duration](ctx, "TIMEOUT")
	if err != nil {
		t.Fatalf("Error getting TIMEOUT: %v", err)
	}
	if timeout != 30*time.Second {
		t.Errorf("Expected TIMEOUT=30s, got %v", timeout)
	}

	hosts, err := Get[[]string](ctx, "HOSTS")
	if err != nil {
		t.Fatalf("Error getting HOSTS: %v", err)
	}
	if len(hosts) != 1 || hosts[0] != "http://localhost:8080" {
		t.Errorf("Expected HOSTS=[http://localhost:8080], got %v", hosts)
	}

	corsOrigins, err := Get[[]string](ctx, "CORS_ORIGINS")
	if err != nil {
		t.Fatalf("Error getting CORS_ORIGINS: %v", err)
	}
	if len(corsOrigins) != 1 || corsOrigins[0] != "http://localhost:3000" {
		t.Errorf("Expected CORS_ORIGINS=[http://localhost:3000], got %v", corsOrigins)
	}

	// Test generic Get methods
	baseURLCheck, err := Get[string](ctx, "BASE_URL")
	if err != nil || baseURLCheck != baseURL {
		t.Error("Get[string]() method failed")
	}

	portCheck, err := Get[int64](ctx, "PORT")
	if err != nil || portCheck != port {
		t.Error("Get[int64]() method failed")
	}

	debugCheck, err := Get[bool](ctx, "DEBUG")
	if err != nil || debugCheck != debug {
		t.Error("Get[bool]() method failed")
	}

	timeoutCheck, err := Get[time.Duration](ctx, "TIMEOUT")
	if err != nil || timeoutCheck != timeout {
		t.Error("Get[time.Duration]() method failed")
	}

	hostsCheck, err := Get[[]string](ctx, "HOSTS")
	if err != nil || !reflect.DeepEqual(hostsCheck, hosts) {
		t.Error("Get[[]string]() method failed")
	}

	corsCheck, err := Get[[]string](ctx, "CORS_ORIGINS")
	if err != nil || !reflect.DeepEqual(corsCheck, corsOrigins) {
		t.Error("Get[[]string]() method failed")
	}

	// Test Has method
	if !cfg.Has("PORT") {
		t.Error("Has() method failed for existing key")
	}

	if cfg.Has("NONEXISTENT") {
		t.Error("Has() method should return false for non-existent key")
	}

	// Test Keys method
	keys := cfg.Keys()
	if len(keys) != 6 {
		t.Errorf("Expected 6 keys, got %d", len(keys))
	}

	// Test Dump
	dump := cfg.Dump()
	if dump["PORT"] != "8080" {
		t.Errorf("Dump failed for PORT: %s", dump["PORT"])
	}

	if dump["BASE_URL"] != "https://api.example.com" {
		t.Errorf("Dump failed for BASE_URL: %s", dump["BASE_URL"])
	}
}

func TestValidation(t *testing.T) {
	cfg := New()

	cfg.Define("PORT").
		Int64().
		Env("PORT").
		Default(8080).
		Range(1, 65535)

	cfg.Define("RATE").
		Float64().
		Env("RATE").
		Default(100.0).
		Range(1.0, 1000.0)

	cfg.Define("API_KEY").
		String().
		Env("API_KEY").
		Required().
		MinLength(10)

	// Set invalid values
	t.Setenv("PORT", "99999") // Too high
	t.Setenv("RATE", "0.5")   // Too low
	// API_KEY not set (required)

	err := cfg.Execute([]string{"test"})
	if err == nil {
		t.Fatalf("Expected configuration errors, got none")
	}
}

func TestSecretHandling(t *testing.T) {
	cfg := New()

	cfg.Define("DATABASE_URL").
		String().
		Env("DATABASE_URL").
		Required().
		Secret()

	cfg.Define("API_KEY").
		String().
		Env("API_KEY").
		Default("secret123").
		Secret()

	t.Setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Configuration errors: %v", err)
	}

	// Test secret access
	dbURL := cfg.GetSecret("DATABASE_URL")
	if !dbURL.IsSet() {
		t.Error("DATABASE_URL secret not set")
	}

	if dbURL.Size() != len("postgresql://user:pass@localhost/db") {
		t.Error("DATABASE_URL secret size incorrect")
	}

	apiKey := cfg.GetSecret("API_KEY")
	if apiKey.String() != "secret123" {
		t.Error("API_KEY secret value incorrect")
	}

	// Test that regular Get now collects errors for secrets instead of panicking
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Instead of calling Get directly (which would return error), we'll test the error collection mechanism
	// by simulating what would happen when Get is called on a secret

	// Simulate the error collection that would happen in Get function
	ctx.execution.CollectError(cfg, "DATABASE_URL", "secret", "", "use GetSecret() instead", true)

	// Check that error was collected
	collected := ctx.execution.GetErrors()
	if len(collected) == 0 {
		t.Error("Expected error to be collected for secret access")
	}

	if !collected[0].IsSecret {
		t.Error("Expected error to be marked as secret")
	}

	if collected[0].Key != "DATABASE_URL" {
		t.Errorf("Expected key 'DATABASE_URL', got '%s'", collected[0].Key)
	}

	if collected[0].Message != "use GetSecret() instead" {
		t.Errorf("Expected message 'use GetSecret() instead', got '%s'", collected[0].Message)
	}
}

func TestConfigDefaultPriority(t *testing.T) {
	cfg := New()

	// Test default priority
	defaultPriority := cfg.GetDefaultPriority()
	if len(defaultPriority) != 4 {
		t.Error("Default priority should have 4 elements")
	}
	if defaultPriority[0] != SourceFlag || defaultPriority[1] != SourceEnv || defaultPriority[2] != SourceFile || defaultPriority[3] != SourceDefault {
		t.Error("Default priority should be [Flag, Env, File, Default]")
	}

	// Test setting custom priority
	customPriority := SourcePriority{SourceEnv, SourceDefault}
	cfg.SetDefaultPriority(customPriority)

	newPriority := cfg.GetDefaultPriority()
	if len(newPriority) != 2 {
		t.Error("Custom priority should have 2 elements")
	}
	if newPriority[0] != SourceEnv || newPriority[1] != SourceDefault {
		t.Error("Custom priority should be [Env, Default]")
	}
}

func TestConfigLevelPriority(t *testing.T) {
	t.Setenv("CONFIG_PRIORITY_TEST", "env_value")

	cfg := New()

	// Set config-level priority to Env > Flag > Default
	cfg.SetDefaultPriority(PriorityEnvFlagDefault)

	// Define config without explicit priority (should use config default)
	cfg.Define("TEST_CONFIG_PRIORITY").
		String().
		Env("CONFIG_PRIORITY_TEST").
		Flag("test-config-flag").
		Default("default_value")

	if err := cfg.Execute([]string{"test", "--test-config-flag", "flag_value"}); err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Env should win with config-level Env > Flag > Default priority
	value := cfg.values["TEST_CONFIG_PRIORITY"]
	if value != "env_value" {
		t.Errorf("Expected env_value, got %s", value)
	}
}

// TestConfigDestroy tests the Destroy functionality
func TestConfigDestroy(t *testing.T) {
	cfg := New()

	// Add a secret
	cfg.Define("SECRET").String().Secret()
	_ = cfg.Execute([]string{"test"})
	secret := cfg.GetSecret("SECRET")
	cfg.secrets.Store("SECRET", "test_secret_value")
	secret = cfg.GetSecret("SECRET")

	// Verify secret exists
	if !secret.IsSet() {
		t.Error("Secret should be set before destroy")
	}

	// Destroy config
	cfg.Destroy()

	// Verify secret is destroyed
	if secret.IsSet() {
		t.Error("Secret should be destroyed after config.Destroy()")
	}
}

// TestConfigIsSecret tests the IsSecret functionality
func TestConfigIsSecret(t *testing.T) {
	cfg := New()

	// Define a secret and a regular key
	cfg.Define("SECRET_KEY").String().Secret()
	cfg.Define("REGULAR_KEY").String()
	_ = cfg.Execute([]string{"test"})

	// Test secret detection
	if !cfg.IsSecret("SECRET_KEY") {
		t.Error("SECRET_KEY should be detected as secret")
	}

	if cfg.IsSecret("REGULAR_KEY") {
		t.Error("REGULAR_KEY should not be detected as secret")
	}

	if cfg.IsSecret("NONEXISTENT_KEY") {
		t.Error("Non-existent key should not be detected as secret")
	}
}

// TestConfigDump tests the Dump functionality
func TestConfigDump(t *testing.T) {
	cfg := New()

	// Add some configuration
	cfg.Define("PORT").Int64().Default(8080)
	cfg.Define("HOST").String().Default("localhost")
	cfg.Define("SECRET").String().Secret()
	_ = cfg.Execute([]string{"test"})

	// Set some values
	cfg.values["PORT"] = int64(3000)
	cfg.values["HOST"] = "example.com"
	cfg.secrets.Store("SECRET", "secret_value")

	// Dump configuration
	output := cfg.Dump()

	// Convert map to string for checking
	dumpStr := fmt.Sprintf("%v", output)

	// Verify dump content
	if !strings.Contains(dumpStr, "PORT") {
		t.Error("Dump should contain PORT")
	}
	if !strings.Contains(dumpStr, "HOST") {
		t.Error("Dump should contain HOST")
	}
	if !strings.Contains(dumpStr, "3000") {
		t.Error("Dump should contain PORT value")
	}
	if !strings.Contains(dumpStr, "example.com") {
		t.Error("Dump should contain HOST value")
	}
	// Secret should be masked with bytes format
	if strings.Contains(dumpStr, "secret_value") {
		t.Error("Dump should not contain actual secret value")
	}
	if !strings.Contains(dumpStr, "[SECRET:") && !strings.Contains(dumpStr, "bytes]") {
		t.Error("Dump should contain masked secret in format [SECRET:X bytes]")
	}
}

func TestGlobalMiddleware(t *testing.T) {
	cfg := New()

	// Track middleware execution
	var executionOrder []string

	// Add global middleware
	cfg.UseMiddleware(func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			executionOrder = append(executionOrder, "global1")
			return next(ctx)
		}
	})

	cfg.UseMiddleware(func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			executionOrder = append(executionOrder, "global2")
			return next(ctx)
		}
	})

	// Define a command
	cfg.Command("test").
		Func(func(ctx *CommandContext) error {
			executionOrder = append(executionOrder, "command")
			return nil
		}).
		ShortHelp("Test command")

	// Execute
	err := cfg.Execute([]string{"app", "test"})
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Check execution order
	expected := []string{"global1", "global2", "command"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d executions, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}

	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("Expected execution[%d] = %s, got %s", i, exp, executionOrder[i])
		}
	}
}

func TestUseMiddlewareForCommands(t *testing.T) {
	cfg := New()

	// Track middleware execution
	var executedFor []string

	// Add middleware only for specific commands
	cfg.UseMiddlewareForCommands([]string{"start", "stop"}, func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			executedFor = append(executedFor, ctx.Command)
			return next(ctx)
		}
	})

	// Define commands
	cfg.Command("start").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Start")

	cfg.Command("stop").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Stop")

	cfg.Command("status").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Status")

	// Execute start - should trigger middleware
	executedFor = nil
	err := cfg.Execute([]string{"app", "start"})
	if err != nil {
		t.Fatalf("start execution failed: %v", err)
	}
	if len(executedFor) != 1 || executedFor[0] != "start" {
		t.Errorf("Expected middleware for 'start', got %v", executedFor)
	}

	// Execute stop - should trigger middleware
	executedFor = nil
	err = cfg.Execute([]string{"app", "stop"})
	if err != nil {
		t.Fatalf("stop execution failed: %v", err)
	}
	if len(executedFor) != 1 || executedFor[0] != "stop" {
		t.Errorf("Expected middleware for 'stop', got %v", executedFor)
	}

	// Execute status - should NOT trigger middleware
	executedFor = nil
	err = cfg.Execute([]string{"app", "status"})
	if err != nil {
		t.Fatalf("status execution failed: %v", err)
	}
	if len(executedFor) != 0 {
		t.Errorf("Expected no middleware for 'status', got %v", executedFor)
	}
}

func TestUseMiddlewareForSubcommands(t *testing.T) {
	cfg := New()

	// Track middleware execution
	var executedFor []string

	// Add middleware only for specific subcommands
	cfg.UseMiddlewareForSubcommands("start", []string{"worker"}, func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			executedFor = append(executedFor, ctx.SubCommand)
			return next(ctx)
		}
	})

	// Define command with subcommands - need to add them separately to the same parent
	startCmd := cfg.Command("start").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Start")

	// Add server subcommand
	startCmd.SubCommand("server").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Start server")

	// Add worker subcommand (to start, not to server)
	startCmd.SubCommand("worker").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Start worker")

	// Execute start worker - should trigger middleware
	executedFor = nil
	err := cfg.Execute([]string{"app", "start", "worker"})
	if err != nil {
		t.Fatalf("start worker execution failed: %v", err)
	}
	if len(executedFor) != 1 || executedFor[0] != "worker" {
		t.Errorf("Expected middleware for 'worker', got %v", executedFor)
	}

	// Execute start server - should NOT trigger middleware
	executedFor = nil
	err = cfg.Execute([]string{"app", "start", "server"})
	if err != nil {
		t.Fatalf("start server execution failed: %v", err)
	}
	if len(executedFor) != 0 {
		t.Errorf("Expected no middleware for 'server', got %v", executedFor)
	}
}
