// cli/flag_parser_test.go
package kli

import (
	"testing"
)

func TestFlagParser_ParseCommand(t *testing.T) {
	flagParser := newFlagParser()

	// Create test definitions
	defs := make(map[string]*Definition)
	defs["port"] = &Definition{
		key:         "port",
		valueType:   TypeInt64,
		flag:        "port",
		description: "HTTP server port",
	}
	defs["verbose"] = &Definition{
		key:         "verbose",
		valueType:   TypeBool,
		flag:        "verbose",
		description: "Enable verbose logging",
	}

	tests := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name: "no flags",
			args: []string{},
			expected: map[string]string{
				"port":    "",
				"verbose": "",
			},
		},
		{
			name: "single flag",
			args: []string{"--port", "8080"},
			expected: map[string]string{
				"port":    "8080",
				"verbose": "",
			},
		},
		{
			name: "multiple flags",
			args: []string{"--port", "3000", "--verbose", "true"},
			expected: map[string]string{
				"port":    "3000",
				"verbose": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedFlags, err := flagParser.ParseCommand(tt.args, defs)

			if err != nil {
				t.Errorf("ParseCommand() returned error: %v", err)
			}

			if parsedFlags == nil {
				t.Fatal("ParseCommand() returned nil")
			}

			// Check parsed values
			for key, expectedValue := range tt.expected {
				if actualValue, exists := parsedFlags.Values[key]; !exists {
					t.Errorf("ParseCommand() missing value for key %s", key)
				} else if *actualValue != expectedValue {
					t.Errorf("ParseCommand() for key %s: expected %s, got %s", key, expectedValue, *actualValue)
				}
			}
		})
	}
}

func TestFlagParser_ParseGlobal(t *testing.T) {
	flagParser := newFlagParser()

	// Create test definitions
	defs := make(map[string]*Definition)
	defs["config"] = &Definition{
		key:         "config",
		valueType:   TypeString,
		flag:        "config",
		description: "Configuration file path",
	}

	tests := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name: "no flags",
			args: []string{},
			expected: map[string]string{
				"config": "",
			},
		},
		{
			name: "with config flag",
			args: []string{"--config", "/path/to/config.yaml"},
			expected: map[string]string{
				"config": "/path/to/config.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedFlags, err := flagParser.ParseGlobal(tt.args, defs)

			if err != nil {
				t.Errorf("ParseGlobal() returned error: %v", err)
			}

			if parsedFlags == nil {
				t.Fatal("ParseGlobal() returned nil")
			}

			// Check parsed values
			for key, expectedValue := range tt.expected {
				if actualValue, exists := parsedFlags.Values[key]; !exists {
					t.Errorf("ParseGlobal() missing value for key %s", key)
				} else if *actualValue != expectedValue {
					t.Errorf("ParseGlobal() for key %s: expected %s, got %s", key, expectedValue, *actualValue)
				}
			}
		})
	}
}

func TestFlagParser_ParseGlobalUnknownFlags(t *testing.T) {
	flagParser := newFlagParser()

	// Create test definitions
	defs := make(map[string]*Definition)
	defs["port"] = &Definition{
		key:         "port",
		valueType:   TypeInt64,
		flag:        "port",
		description: "HTTP server port",
	}

	// Undefined flags (e.g. go test flags) are reported as parse errors,
	// but defined flags are still parsed.
	args := []string{"--port", "8080", "-test.timeout", "30s"}

	parsedFlags, err := flagParser.ParseGlobal(args, defs)

	if err != nil {
		t.Errorf("ParseGlobal() returned error: %v", err)
	}

	if parsedFlags == nil {
		t.Fatal("ParseGlobal() returned nil")
	}

	// Check that port was parsed
	if portValue, exists := parsedFlags.Values["port"]; !exists {
		t.Error("ParseGlobal() missing value for key port")
	} else if *portValue != "8080" {
		t.Errorf("ParseGlobal() for port: expected 8080, got %s", *portValue)
	}

	// Undefined flags are collected in Errors, not silently filtered
	if len(parsedFlags.Errors) == 0 {
		t.Error("ParseGlobal() should collect errors for undefined flags")
	}

	// Undefined flags never appear in parsed values
	if _, exists := parsedFlags.Values["test.timeout"]; exists {
		t.Error("ParseGlobal() should not produce values for undefined flags")
	}
}

func TestParsedFlags_Structure(t *testing.T) {
	flagParser := newFlagParser()

	defs := make(map[string]*Definition)
	defs["port"] = &Definition{
		key:         "port",
		valueType:   TypeInt64,
		flag:        "port",
		description: "HTTP server port",
	}

	args := []string{"--port", "8080"}
	parsedFlags, err := flagParser.ParseCommand(args, defs)

	if err != nil {
		t.Errorf("ParseCommand() returned error: %v", err)
	}

	if parsedFlags == nil {
		t.Fatal("ParseCommand() returned nil")
	}

	// Check ParsedFlags structure
	if parsedFlags.Values == nil {
		t.Error("ParsedFlags.Values should not be nil")
	}
	if parsedFlags.FlagSet == nil {
		t.Error("ParsedFlags.FlagSet should not be nil")
	}
}
