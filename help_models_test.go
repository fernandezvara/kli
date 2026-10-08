// cli/help_models_test.go
package kli

import (
	"testing"
)

func TestCommandSummary(t *testing.T) {
	summary := commandSummary{
		Name:    "deploy",
		Aliases: []string{"dep", "deploy-app"},
	}

	if summary.Name != "deploy" {
		t.Errorf("Expected name 'deploy', got '%s'", summary.Name)
	}

	if len(summary.Aliases) != 2 {
		t.Errorf("Expected 2 aliases, got %d", len(summary.Aliases))
	}

	if summary.Aliases[0] != "dep" {
		t.Errorf("Expected first alias 'dep', got '%s'", summary.Aliases[0])
	}
}

func TestFlagInfo(t *testing.T) {
	flag := flagInfo{
		DisplayLine: "--port int64",
		Required:    true,
	}

	if flag.DisplayLine != "--port int64" {
		t.Errorf("Expected display line '--port int64', got '%s'", flag.DisplayLine)
	}

	if !flag.Required {
		t.Error("Expected flag to be required")
	}
}

func TestSubcommandInfo(t *testing.T) {
	info := subcommandInfo{
		Name: "server",
	}

	if info.Name != "server" {
		t.Errorf("Expected subcommand name 'server', got '%s'", info.Name)
	}
}

func TestArgsContainFullHelp(t *testing.T) {
	tests := []struct {
		args     []string
		expected bool
	}{
		{[]string{}, false},
		{[]string{"--help"}, false},
		{[]string{"--full-help"}, true},
		{[]string{"command", "--full-help"}, true},
		{[]string{"command", "subcommand", "--full-help"}, true},
		{[]string{"--full-help", "other"}, true},
	}

	for _, test := range tests {
		result := argsContainFullHelp(test.args)
		if result != test.expected {
			t.Errorf("argsContainFullHelp(%v) = %v, expected %v", test.args, result, test.expected)
		}
	}
}
