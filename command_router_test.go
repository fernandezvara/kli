// cli/command_router_test.go
package kli

import (
	"strings"
	"testing"
)

func TestConfigExecuteUnknownCommand(t *testing.T) {
	cfg := New()
	cfg.Command("start").Func(startCommand).ShortHelp("Start")

	err := cfg.Execute([]string{"app", "stat"})
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !strings.Contains(err.Error(), `unknown command: "stat"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "Did you mean: start?") {
		t.Fatalf("expected suggestion in error: %v", err)
	}
}

// TestTopLevelAliasRouting verifies that registered command aliases resolve
// to the aliased command at the top level of routing.
func TestTopLevelAliasRouting(t *testing.T) {
	cfg := New()
	executed := false

	cfg.Command("run").
		Func(func(ctx *CommandContext) error {
			executed = true
			return nil
		}).
		ShortHelp("Run the task").
		Aliases("r", "exec")

	// Route via alias "r"
	router := newCommandRouter()
	cmd, ctx, err := router.RouteWithHelpHandling([]string{"test", "r"}, cfg)
	if err != nil {
		t.Fatalf("Routing via alias 'r' failed: %v", err)
	}
	if cmd == nil {
		t.Fatal("Expected non-nil command when routing via alias 'r'")
	}
	if ctx.Command != "run" {
		t.Errorf("Expected resolved command name 'run', got %q", ctx.Command)
	}

	// Execute through the full config path using the second alias
	if err := cfg.Execute([]string{"test", "exec"}); err != nil {
		t.Fatalf("Execute via alias 'exec' failed: %v", err)
	}
	if !executed {
		t.Error("Expected aliased command to execute")
	}
}
