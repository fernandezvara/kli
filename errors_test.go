package kli

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfigErrorString(t *testing.T) {
	tests := []struct {
		err      ConfigError
		expected string
	}{
		{
			ConfigError{Key: "PORT", Source: "env", Value: "8080", Display: "", ErrorDescription: "invalid value"},
			"invalid value",
		},
		{
			ConfigError{Key: "HOST", Source: "flag", Value: "", Display: "", ErrorDescription: "required"},
			"required",
		},
		{
			ConfigError{Key: "API_KEY", Source: "none", Value: "", Display: "", ErrorDescription: "not provided"},
			"not provided",
		},
	}

	for _, tt := range tests {
		result := tt.err.Error()
		if result != tt.expected {
			t.Errorf("ConfigError.Error() = %q, expected %q", result, tt.expected)
		}
	}
}

func TestShouldDisplayDefault(t *testing.T) {
	tests := []struct {
		name     string
		def      *Definition
		expected bool
	}{
		{
			name:     "nil default",
			def:      &Definition{valueType: TypeString, defaultValue: nil},
			expected: false,
		},
		{
			name:     "bool false default hidden",
			def:      &Definition{valueType: TypeBool, defaultValue: false},
			expected: false,
		},
		{
			name:     "bool true default shown",
			def:      &Definition{valueType: TypeBool, defaultValue: true},
			expected: true,
		},
		{
			name:     "string default shown",
			def:      &Definition{valueType: TypeString, defaultValue: "info"},
			expected: true,
		},
	}

	for _, tt := range tests {
		if got := shouldDisplayDefault(tt.def); got != tt.expected {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.expected, got)
		}
	}
}

func TestBuildErrorDisplay(t *testing.T) {
	flagDef := &Definition{
		key:          "PORT",
		valueType:    TypeInt64,
		flag:         "port",
		defaultValue: int64(8080),
		required:     true,
	}

	if got := buildErrorDisplay(flagDef); got != "--port int64 (default: 8080, required)" {
		t.Fatalf("unexpected flag error display: %q", got)
	}

	envDef := &Definition{
		key:       "DATABASE_URL",
		valueType: TypeString,
		envVar:    "DATABASE_URL",
		required:  true,
	}

	if got := buildErrorDisplay(envDef); got != "DATABASE_URL string (required)" {
		t.Fatalf("unexpected env-only error display: %q", got)
	}
}

func TestBuildDefinitionDisplay(t *testing.T) {
	def := &Definition{
		key:          "LOG_LEVEL",
		valueType:    TypeString,
		flag:         "log-level",
		envVar:       "LOG_LEVEL",
		defaultValue: "info",
		validations: []Validation{
			{Name: "oneOf(debug,info,warn,error)"},
		},
	}

	got := buildDefinitionDisplay(def)
	expectedParts := []string{
		"--log-level string",
		"(default: info, oneOf: ['debug', 'info', 'warn', 'error'], env: LOG_LEVEL)",
	}

	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Fatalf("expected %q in %q", part, got)
		}
	}
}

func TestNewConfigErrorConstructors(t *testing.T) {
	def := &Definition{
		key:          "PORT",
		valueType:    TypeInt64,
		flag:         "port",
		defaultValue: int64(8080),
	}

	validationErr := newConfigError("PORT", def, "flag", "80000", fmt.Errorf("value 80000 is greater than maximum 65535"))
	if validationErr.Display != "--port int64 (default: 8080)" {
		t.Fatalf("unexpected validation display: %q", validationErr.Display)
	}
	if validationErr.ErrorDescription != "value 80000 is greater than maximum 65535" {
		t.Fatalf("unexpected validation error description: %q", validationErr.ErrorDescription)
	}

	parseErr := newConfigError("PORT", def, "flag", "abc", fmt.Errorf("invalid syntax"))
	if parseErr.ErrorDescription != "invalid syntax" {
		t.Fatalf("unexpected parse error description: %q", parseErr.ErrorDescription)
	}

	requiredDef := &Definition{
		key:       "BASE_URL",
		valueType: TypeString,
		flag:      "base-url",
		required:  true,
	}
	requiredErr := newConfigError("BASE_URL", requiredDef, "validation", "", fmt.Errorf("Not provided"))
	if requiredErr.Display != "--base-url string (required)" {
		t.Fatalf("unexpected required display: %q", requiredErr.Display)
	}
	if requiredErr.ErrorDescription != "Not provided" {
		t.Fatalf("unexpected required error description: %q", requiredErr.ErrorDescription)
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"a", "****"},
		{"ab", "****"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcde", "ab*de"},
		{"abcdef", "ab**ef"},
		{"secret-key-12345", "se************45"},
		{"my-super-secret-api-key", "my*******************ey"},
	}

	for _, tt := range tests {
		result := maskSecret(tt.input)
		if result != tt.expected {
			t.Errorf("maskSecret(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskSecretLength(t *testing.T) {
	// For strings longer than 4 chars, masked length should equal original length
	inputs := []string{"hello", "password123", "super-secret-key"}

	for _, input := range inputs {
		result := maskSecret(input)
		if len(result) != len(input) {
			t.Errorf("maskSecret(%q) length = %d, expected %d", input, len(result), len(input))
		}
	}
}
