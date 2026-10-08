// cli/help_service_test.go
package kli

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCommandHelp(t *testing.T) {
	cfg := New()

	// Define a command with help
	cfg.Command("start").
		Func(startCommand).
		ShortHelp("Start the service").
		LongHelp("This is a detailed help text for the start command.\nIt explains how to use the command.").
		Config(func(cc *CommandConfig) {
			cc.Define("PORT").Int64().Flag("port").Default(8080).Description("Server port")
			cc.Define("DAEMON").Bool().Flag("daemon").Default(false).Description("Run in background")
		})

	// Use new help system to get command help
	helpService := newHelpService()
	help, err := helpService.GenerateHelp([]string{"app", "start", "--help"}, cfg.commands)
	if err != nil {
		t.Fatalf("Failed to generate help: %v", err)
	}

	// Check that help contains expected content
	if !contains(help, "This is a detailed help text") {
		t.Error("Long help not found in help text")
	}

	if !contains(help, "Server port") {
		t.Error("PORT description not found in help text")
	}

	if !contains(help, "--port") {
		t.Error("Port flag not found in help text")
	}

	if !contains(help, "Run in background") {
		t.Error("DAEMON description not found in help text")
	}
}

func TestShowGlobalHelp(t *testing.T) {
	cfg := New()
	cfg.Command("start").Func(startCommand).LongHelp("Start the service").Aliases("run")
	cfg.Command("stop").Func(stopCommand).LongHelp("Stop the service")

	output := captureStdout(t, func() {
		if err := cfg.ShowGlobalHelp(); err != nil {
			t.Fatalf("ShowGlobalHelp failed: %v", err)
		}
	})

	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected usage in output, got: %s", output)
	}
	if !strings.Contains(output, "Available commands:") {
		t.Fatalf("expected commands heading in output, got: %s", output)
	}
	if !strings.Contains(output, "start") || !strings.Contains(output, "Start the service") {
		t.Fatalf("expected start command in output, got: %s", output)
	}
	if !strings.Contains(output, "stop") || !strings.Contains(output, "Stop the service") {
		t.Fatalf("expected stop command in output, got: %s", output)
	}
	if !strings.Contains(output, "aliases: run") {
		t.Fatalf("expected aliases in output, got: %s", output)
	}
	if !strings.Contains(output, "<command> --help") {
		t.Fatalf("expected command help hint in output, got: %s", output)
	}
}

func TestShowCommandHelp(t *testing.T) {
	cfg := New()
	cfg.Command("deploy").
		Func(testCommand).
		ShortHelp("Deploy the service").
		LongHelp("Deploys the current release.").
		Config(func(cc *CommandConfig) {
			cc.Define("ENV").String().Flag("env").Required().Description("Target environment")
			cc.Define("FORCE").Bool().Flag("force").Default(false).Description("Force deployment")
		})

	output := captureStdout(t, func() {
		if err := cfg.ShowCommandHelp("deploy"); err != nil {
			t.Fatalf("ShowCommandHelp failed: %v", err)
		}
	})

	if !strings.Contains(output, "Usage:") || !strings.Contains(output, "deploy [options]") {
		t.Fatalf("expected deploy usage in output, got: %s", output)
	}
	if !strings.Contains(output, "Deploys the current release.") {
		t.Fatalf("expected long help in output, got: %s", output)
	}
	if !strings.Contains(output, "--env") || !strings.Contains(output, "Target environment") {
		t.Fatalf("expected env option in output, got: %s", output)
	}
	if !strings.Contains(output, "--force") || !strings.Contains(output, "Force deployment") {
		t.Fatalf("expected force option in output, got: %s", output)
	}
}

func TestShowCommandHelpUnknownCommand(t *testing.T) {
	cfg := New()

	// The unified system shows help for unknown commands without error
	err := cfg.ShowCommandHelp("missing")
	if err != nil {
		t.Fatalf("unexpected error for unknown command: %v", err)
	}
	// This should succeed and show help for the unknown command
}

func TestConfigExecuteHelpPaths(t *testing.T) {
	cfg := New()
	cfg.Command("deploy").Func(testCommand).ShortHelp("Deploy service")

	globalOutput := captureStdout(t, func() {
		if err := cfg.Execute([]string{"app"}); err != nil {
			t.Fatalf("Execute without command failed: %v", err)
		}
	})

	if !strings.Contains(globalOutput, "Available commands:") || !strings.Contains(globalOutput, "deploy") {
		t.Fatalf("expected global help output, got: %s", globalOutput)
	}

	commandOutput := captureStdout(t, func() {
		if err := cfg.Execute([]string{"app", "help", "deploy"}); err != nil {
			t.Fatalf("Execute help deploy failed: %v", err)
		}
	})

	if !strings.Contains(commandOutput, "Deploy service") || !strings.Contains(commandOutput, "Usage:") {
		t.Fatalf("expected command help output, got: %s", commandOutput)
	}
}

func TestGetSubcommandHelp(t *testing.T) {
	cmd := &Command{
		Name:        "user",
		ShortHelp:   "User management commands",
		SubCommands: make(map[string]*Command),
	}

	// Add subcommands
	cmd.SubCommands["create"] = &Command{
		Name:      "create",
		ShortHelp: "Create a new user",
	}
	cmd.SubCommands["update"] = &Command{
		Name:      "update",
		ShortHelp: "Update an existing user",
	}
	cmd.SubCommands["list"] = &Command{
		Name:      "list",
		ShortHelp: "List all users",
	}
	cmd.SubCommands["show"] = &Command{
		Name:      "show",
		ShortHelp: "Show details of a user",
	}
	cmd.SubCommands["delete"] = &Command{
		Name:      "delete",
		ShortHelp: "Delete a user",
	}

	// Use new help system to get subcommand help
	helpService := newHelpService()
	commands := map[string]*Command{"user": cmd}
	help, err := helpService.GenerateHelp([]string{"user", "--help"}, commands)
	if err != nil {
		t.Fatalf("Failed to generate help: %v", err)
	}

	// Check help content - updated for new help system format
	expectedParts := []string{
		"user [options]",
		"User management commands",
		"Subcommands:",
		"create       Create a new user",
		"update       Update an existing user",
		"list         List all users",
		"show         Show details of a user",
		"delete       Delete a user",
	}

	for _, part := range expectedParts {
		if !strings.Contains(help, part) {
			t.Errorf("Expected '%s' in help, got: %s", part, help)
		}
	}

	// Print the actual help output to verify format
	t.Logf("Actual help output:\n%s", help)
}

// TestConfigGenerateHelp tests the GenerateHelp functionality with new help system
func TestConfigGenerateHelp(t *testing.T) {
	cfg := New()

	// Add a command to test command help generation
	cfg.Command("start").Func(func(ctx *CommandContext) error {
		return nil
	}).ShortHelp("Start the service")

	// Add some configuration
	cfg.Define("PORT").Int64().Default(8080).Description("Server port")
	cfg.Define("HOST").String().Default("localhost").Description("Server host")
	_ = cfg.Execute([]string{"test"})

	// Generate help (now shows command help instead of config help)
	help := cfg.GenerateHelp()

	// Verify help content shows command help format
	if !strings.Contains(help, "Available commands") {
		t.Error("Help should contain 'Available commands'")
	}
	if !strings.Contains(help, "start") {
		t.Error("Help should contain 'start' command")
	}
}

func TestHelpWithCustomValidationAndEnvironment(t *testing.T) {
	// Test that help works even when environment variables have validation issues

	// Set an environment variable with a value that would fail custom validation
	t.Setenv("TEST_PORT", "invalid_port")

	cfg := New()

	// Define a custom validator that expects int64
	portValidator := func(value any) error {
		if port, ok := value.(int64); ok {
			if port < 1 || port > 65535 {
				return fmt.Errorf("port must be between 1 and 65535, got %d", port)
			}
			return nil
		}
		return fmt.Errorf("port must be an integer, got %T", value)
	}

	// Define configuration with custom validation
	cfg.Define("PORT").
		Int64().
		Env("TEST_PORT").
		Flag("port").
		Default(8080).
		Custom("port_range", portValidator).
		Description("Server port")

	// Define another with string validation
	stringValidator := func(value any) error {
		if s, ok := value.(string); ok {
			if len(s) < 3 {
				return fmt.Errorf("string must be at least 3 characters, got %d", len(s))
			}
			return nil
		}
		return fmt.Errorf("value must be a string, got %T", value)
	}

	cfg.Define("NAME").
		String().
		Env("TEST_NAME").
		Flag("name").
		Default("default").
		Custom("min_length", stringValidator).
		Description("Application name")

	// Test that help works despite invalid environment variable
	args := []string{"cmd", "test", "--help"}
	err := cfg.Execute(args)

	// Help should work without errors
	if err != nil {
		t.Errorf("Help should work despite invalid environment variable, got error: %v", err)
	}
}

func TestNormalExecutionWithCustomValidation(t *testing.T) {
	// Test that normal execution still validates properly

	// Set environment variables with valid values
	t.Setenv("TEST_PORT", "3000")
	t.Setenv("TEST_NAME", "valid_name")

	cfg := New()

	// Define a custom validator that expects int64
	portValidator := func(value any) error {
		if port, ok := value.(int64); ok {
			if port < 1 || port > 65535 {
				return fmt.Errorf("port must be between 1 and 65535, got %d", port)
			}
			return nil
		}
		return fmt.Errorf("port must be an integer, got %T", value)
	}

	// Define configuration with custom validation
	cfg.Define("PORT").
		Int64().
		Env("TEST_PORT").
		Flag("port").
		Default(8080).
		Custom("port_range", portValidator).
		Description("Server port")

	// Define another with string validation
	stringValidator := func(value any) error {
		if s, ok := value.(string); ok {
			if len(s) < 3 {
				return fmt.Errorf("string must be at least 3 characters, got %d", len(s))
			}
			return nil
		}
		return fmt.Errorf("value must be a string, got %T", value)
	}

	cfg.Define("NAME").
		String().
		Env("TEST_NAME").
		Flag("name").
		Default("default").
		Custom("min_length", stringValidator).
		Description("Application name")

	// Add a simple command
	cfg.Command("test").
		Func(func(ctx *CommandContext) error {
			// Try to get the values - this should work with proper types
			port, err := Get[int64](ctx, "PORT")
			if err != nil {
				return fmt.Errorf("failed to get PORT: %w", err)
			}

			name, err := Get[string](ctx, "NAME")
			if err != nil {
				return fmt.Errorf("failed to get NAME: %w", err)
			}

			// Verify types are correct
			if port != 3000 {
				t.Errorf("Expected port 3000, got %d (type: %T)", port, port)
			}

			if name != "valid_name" {
				t.Errorf("Expected name 'valid_name', got %s (type: %T)", name, name)
			}

			return nil
		}).
		ShortHelp("Test command")

	// Test normal execution - should work with valid environment variables
	args := []string{"cmd", "test"}
	err := cfg.Execute(args)

	if err != nil {
		t.Errorf("Normal execution should work with valid environment variables, got error: %v", err)
	}
}

func TestHelpWithDifferentTypesAndCustomValidation(t *testing.T) {
	// Test help with various types and custom validators

	cfg := New()

	// Bool custom validator
	boolValidator := func(value any) error {
		if _, ok := value.(bool); ok {
			return nil
		}
		return fmt.Errorf("value must be boolean, got %T", value)
	}

	// Duration custom validator
	durationValidator := func(value any) error {
		if d, ok := value.(time.Duration); ok {
			if d < 0 {
				return fmt.Errorf("duration must be positive, got %v", d)
			}
			return nil
		}
		return fmt.Errorf("value must be duration, got %T", value)
	}

	// Float64 custom validator
	floatValidator := func(value any) error {
		if f, ok := value.(float64); ok {
			if f < 0 {
				return fmt.Errorf("value must be positive, got %f", f)
			}
			return nil
		}
		return fmt.Errorf("value must be float, got %T", value)
	}

	// Define configurations with different types and custom validation
	cfg.Define("ENABLED").
		Bool().
		Env("TEST_ENABLED").
		Flag("enabled").
		Default(false).
		Custom("bool_check", boolValidator).
		Description("Enable feature")

	cfg.Define("TIMEOUT").
		Duration().
		Env("TEST_TIMEOUT").
		Flag("timeout").
		Default(30*time.Second).
		Custom("positive_duration", durationValidator).
		Description("Operation timeout")

	cfg.Define("RATE").
		Float64().
		Env("TEST_RATE").
		Flag("rate").
		Default(1.0).
		Custom("positive_float", floatValidator).
		Description("Rate limit")

	// Add a command
	cfg.Command("test").
		Func(func(ctx *CommandContext) error {
			return nil
		}).
		ShortHelp("Test command")

	// Test that help works for all types
	args := []string{"cmd", "test", "--help"}
	err := cfg.Execute(args)

	// Help should work without errors
	if err != nil {
		t.Errorf("Help should work with all types and custom validation, got error: %v", err)
	}
}
