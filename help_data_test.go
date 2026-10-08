// cli/help_data_test.go
package kli

import (
	"testing"
)

func TestNewUnifiedExtractor(t *testing.T) {
	extractor := newUnifiedExtractor()

	if extractor == nil {
		t.Fatal("Expected non-nil extractor")
	}
}

func TestUnifiedExtractor_ExtractFlags(t *testing.T) {
	extractor := newUnifiedExtractor()

	defs := map[string]*Definition{
		"flag": {
			key:         "flag",
			flag:        "flag",
			valueType:   TypeString,
			required:    true,
			description: "A flag",
		},
		"env": {
			key:         "env",
			envVar:      "ENV",
			valueType:   TypeString,
			required:    false,
			description: "An env var",
		},
	}

	flags := extractor.ExtractFlags(defs)

	if len(flags) != 1 {
		t.Errorf("Expected 1 flag (only actual flags), got %d", len(flags))
	}

	if flags[0].DisplayLine == "" {
		t.Error("Expected flag to have DisplayLine")
	}

	// Environment variables should be extracted separately using ExtractEnvVars
	envVars := extractor.ExtractEnvVars(defs)
	if len(envVars) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(envVars))
	}
}

func TestUnifiedExtractor_FilterEnvVars(t *testing.T) {
	extractor := newUnifiedExtractor()

	flags := []flagInfo{
		{
			EnvVar:   "",
			Required: true,
		},
		{
			EnvVar:   "REQUIRED_ENV",
			Required: true,
		},
		{
			EnvVar:   "OPTIONAL_ENV",
			Required: false,
		},
	}

	// Test essential mode - only required env vars
	envVars := extractor.FilterEnvVars(flags, helpModeEssential)

	if len(envVars) != 1 {
		t.Errorf("Expected 1 env var in essential mode, got %d", len(envVars))
	}

	if envVars[0].EnvVar != "REQUIRED_ENV" {
		t.Error("Expected to find only required env var in essential mode")
	}

	// Test full mode - all env vars
	envVars = extractor.FilterEnvVars(flags, helpModeFull)

	if len(envVars) != 2 {
		t.Errorf("Expected 2 env vars in full mode, got %d", len(envVars))
	}
}

func TestUnifiedExtractor_ExtractSubcommands(t *testing.T) {
	extractor := newUnifiedExtractor()

	cmd := &Command{
		SubCommands: map[string]*Command{
			"sub1": {
				Name:     "sub1",
				LongHelp: "Subcommand 1",
				Aliases:  []string{"alias1"},
			},
			"sub2": {
				Name:     "sub2",
				LongHelp: "Subcommand 2",
				Aliases:  []string{"alias2", "alias3"},
			},
		},
	}

	subcommands := extractor.ExtractSubcommands(cmd)

	if len(subcommands) != 2 {
		t.Errorf("Expected 2 subcommands, got %d", len(subcommands))
	}

	// Check that subcommands are sorted
	if subcommands[0].Name != "sub1" || subcommands[1].Name != "sub2" {
		t.Error("Expected subcommands to be sorted by name")
	}

	// Check subcommand 1
	if subcommands[0].Description != "Subcommand 1" {
		t.Error("Expected subcommand 1 to have correct description")
	}

	if len(subcommands[0].Aliases) != 1 || subcommands[0].Aliases[0] != "alias1" {
		t.Error("Expected subcommand 1 to have correct aliases")
	}
}

func TestUnifiedExtractor_ExtractGlobalCommands(t *testing.T) {
	extractor := newUnifiedExtractor()

	commands := map[string]*Command{
		"cmd1": {
			Name:     "cmd1",
			LongHelp: "Command 1",
			Aliases:  []string{"alias1"},
		},
		"cmd2": {
			Name:     "cmd2",
			LongHelp: "Command 2",
			Aliases:  []string{},
		},
	}

	commandsData := extractor.extractCommandsData(commands, "testapp")
	summaries := commandsData.commands

	if len(summaries) != 2 {
		t.Errorf("Expected 2 command summaries, got %d", len(summaries))
	}

	// Check that commands are sorted
	if summaries[0].Name != "cmd1" || summaries[1].Name != "cmd2" {
		t.Error("Expected commands to be sorted by name")
	}

	// Check command 1
	if summaries[0].Description != "Command 1" {
		t.Error("Expected command 1 to have correct description")
	}

	if len(summaries[0].Aliases) != 1 || summaries[0].Aliases[0] != "alias1" {
		t.Error("Expected command 1 to have correct aliases")
	}
}

func TestFlagInfo_GetDisplayLine(t *testing.T) {
	// Test flag
	flag := flagInfo{
		DisplayLine: "--flag string",
	}

	if flag.DisplayLine != "--flag string" {
		t.Errorf("Expected '--flag string', got '%s'", flag.DisplayLine)
	}

	// Test environment variable
	env := flagInfo{
		EnvVar:        "ENV_VAR",
		EnvVarDisplay: "ENV_VAR string",
	}

	if env.EnvVarDisplay != "ENV_VAR string" {
		t.Errorf("Expected 'ENV_VAR string', got '%s'", env.EnvVarDisplay)
	}
}
