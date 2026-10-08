// cli/command_test.go
package kli

import (
	"testing"
)

func TestCommandDefinition(t *testing.T) {
	cfg := New()

	// Define a simple command
	cfg.Command("start").
		Func(startCommand).
		ShortHelp("Start the service").
		LongHelp("Start the service with all components initialized").
		Aliases("s", "run").
		Config(func(cc *CommandConfig) {
			cc.Define("PORT").Int64().Flag("port").Default(8080).Range(1, 65535)
			cc.Define("DAEMON").Bool().Flag("daemon").Default(false)
		})

	// Check command was registered
	cmd, exists := cfg.commands["start"]
	if !exists {
		t.Fatal("Command not registered")
	}

	if cmd.Name != "start" {
		t.Errorf("Expected command name 'start', got '%s'", cmd.Name)
	}

	if cmd.ShortHelp != "Start the service" {
		t.Errorf("Expected short help 'Start the service', got '%s'", cmd.ShortHelp)
	}

	if cmd.LongHelp != "Start the service with all components initialized" {
		t.Errorf("Expected long help 'Start the service with all components initialized', got '%s'", cmd.LongHelp)
	}

	if len(cmd.Aliases) != 2 || cmd.Aliases[0] != "s" || cmd.Aliases[1] != "run" {
		t.Errorf("Expected aliases ['s', 'run'], got %v", cmd.Aliases)
	}

	// Check command-specific configuration
	if len(cmd.Definitions) != 2 {
		t.Errorf("Expected 2 command definitions, got %d", len(cmd.Definitions))
	}

	if portDef, exists := cmd.Definitions["PORT"]; !exists {
		t.Error("PORT definition not found in command")
	} else if portDef.defaultValue != int64(8080) {
		t.Errorf("Expected PORT default 8080, got %v", portDef.defaultValue)
	}
}

func TestCommandExecution(t *testing.T) {
	cfg := New()

	// Define a command
	cfg.Command("echo").
		Func(echoCommand).
		ShortHelp("Echo arguments").
		Config(func(cc *CommandConfig) {
			cc.Define("UPPERCASE").Bool().Flag("uppercase").Default(false)
		})

	// Test execution
	ctx := NewCommandContext([]string{"hello", "world"}, cfg, "echo", "")

	result := cfg.commands["echo"].Execute(ctx)
	if result.Error != nil {
		t.Fatalf("Command execution failed: %v", result.Error)
	}
}

func TestSubCommands(t *testing.T) {
	cfg := New()

	// Define a command with subcommands
	cfg.Command("start").
		Func(startCommand).
		ShortHelp("Start the service").
		SubCommand("server").
		Func(startServerCommand).
		ShortHelp("Start server only").
		Aliases("srv").
		SubCommand("worker").
		Func(startWorkerCommand).
		ShortHelp("Start worker only").
		Aliases("wrk")

	// Test subcommand finding
	cmd := cfg.commands["start"]

	serverCmd := cmd.FindSubCommand("server")
	if serverCmd == nil {
		t.Fatal("Server subcommand not found")
	}

	if serverCmd.ShortHelp != "Start server only" {
		t.Errorf("Expected server short help 'Start server only', got '%s'", serverCmd.ShortHelp)
	}

	// Test alias finding
	serverCmd2 := cmd.FindSubCommand("srv")
	if serverCmd2 == nil {
		t.Fatal("Server subcommand not found by alias")
	}

	// Test non-existent subcommand
	nonExistent := cmd.FindSubCommand("nonexistent")
	if nonExistent != nil {
		t.Error("Should return nil for non-existent subcommand")
	}
}

func TestCommandSuggestions(t *testing.T) {
	cfg := New()

	// Define some commands
	cfg.Command("start").Func(startCommand).ShortHelp("Start")
	cfg.Command("stop").Func(stopCommand).ShortHelp("Stop")
	cfg.Command("restart").Func(restartCommand).ShortHelp("Restart")

	// Test suggestions
	suggestions := cfg.findSuggestions("stat")
	if !contains(suggestions, "start") {
		t.Error("Should suggest 'start' for 'stat'")
	}

	suggestions = cfg.findSuggestions("stp")
	if !contains(suggestions, "stop") {
		t.Error("Should suggest 'stop' for 'stp'")
	}

	suggestions = cfg.findSuggestions("restart")
	if !contains(suggestions, "restart") {
		t.Error("Should suggest 'restart' for exact match")
	}

	// Test non-matching
	suggestions = cfg.findSuggestions("xyz")
	if suggestions != "no similar commands found" {
		t.Errorf("Expected 'no similar commands found' for 'xyz', got '%s'", suggestions)
	}
}

func TestCommandMiddleware(t *testing.T) {
	cfg := New()

	// Define a command with middleware
	cfg.Command("test").
		Func(testCommand).
		ShortHelp("Test command").
		Middleware(loggingMiddleware).
		Middleware(authMiddleware)

	cmd := cfg.commands["test"]

	if len(cmd.Middleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(cmd.Middleware))
	}
}

func TestCommandWithNilFuncAndSubcommands(t *testing.T) {
	cfg := New()

	// Define a command with subcommands but no Func
	cfg.Command("user").
		ShortHelp("User management commands").
		SubCommand("create").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Create a new user").
		SubCommand("update").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Update an existing user").
		SubCommand("list").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("List all users").
		SubCommand("show").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Show details of a user").
		SubCommand("delete").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Delete a user")

	// Test executing parent command without subcommand - should show help and succeed
	err := cfg.Execute([]string{"app", "user"})
	if err != nil {
		t.Fatalf("Expected no error when executing command with subcommands (help should be shown), got %v", err)
	}

	// With new help system, help is shown directly and no error is returned
	// The test passes if we get here without an error
}

func TestCommandWithNilFuncAndNoSubcommands(t *testing.T) {
	cfg := New()

	// Define a command with no Func and no subcommands
	cfg.Command("broken").
		ShortHelp("This command has no implementation")

	// Test executing command with no Func and no subcommands
	err := cfg.Execute([]string{"app", "broken"})
	if err == nil {
		t.Fatal("Expected error when executing command with nil Func and no subcommands")
	}

	expectedErr := "command 'broken' has no implementation"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestCommandWithFuncAndSubcommands(t *testing.T) {
	cfg := New()
	executed := false

	// Define a command with both Func and subcommands (should work normally)
	cfg.Command("start").
		Func(func(ctx *CommandContext) error {
			executed = true
			return nil
		}).
		ShortHelp("Start the service").
		SubCommand("server").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Start server only")

	// Test executing parent command - should execute Func normally
	err := cfg.Execute([]string{"app", "start"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !executed {
		t.Error("Expected command Func to be executed")
	}
}
