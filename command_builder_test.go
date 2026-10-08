// cli/command_builder_test.go
package kli

import (
	"testing"
)

// TestSubCommandBuilder tests CommandBuilder.SubCommand
func TestSubCommandBuilder(t *testing.T) {
	cfg := New()

	cfg.Command("parent").
		Func(func(ctx *CommandContext) error { return nil }).
		SubCommand("child").
		Func(func(ctx *CommandContext) error { return nil })

	// Verify subcommand was added
	parent := cfg.commands["parent"]
	if parent == nil {
		t.Fatal("parent command not registered")
	}
	if len(parent.SubCommands) != 1 {
		t.Errorf("Expected 1 subcommand, got %d", len(parent.SubCommands))
	}

	if found, exists := parent.SubCommands["child"]; !exists {
		t.Error("Subcommand 'child' not found")
	} else if found.Func == nil {
		t.Error("Subcommand 'child' has no Func")
	}
}
