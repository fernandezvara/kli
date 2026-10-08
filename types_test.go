package kli

import (
	"testing"
	"time"
)

func TestValueTypeString(t *testing.T) {
	tests := []struct {
		vt       ValueType
		expected string
	}{
		{TypeString, "string"},
		{TypeInt64, "int64"},
		{TypeFloat64, "float64"},
		{TypeBool, "bool"},
		{TypeDuration, "duration"},
		{TypeStringSlice, "[]string"},
		{TypeInt64Slice, "[]int64"},
		{ValueType(99), "unknown"},
	}

	for _, tt := range tests {
		result := tt.vt.String()
		if result != tt.expected {
			t.Errorf("ValueType(%d).String() = %s, expected %s", tt.vt, result, tt.expected)
		}
	}
}

func TestParseValueString(t *testing.T) {
	result, err := parseValue("hello", TypeString, ",")
	if err != nil {
		t.Fatalf("parseValue string failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got %v", result)
	}

	// Empty string
	result, err = parseValue("", TypeString, ",")
	if err != nil {
		t.Fatalf("parseValue empty string failed: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for empty string, got %v", result)
	}
}

func TestParseValueInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"123", 123, false},
		{"-456", -456, false},
		{"0", 0, false},
		{"9223372036854775807", 9223372036854775807, false}, // max int64
		{"invalid", 0, true},
		{"12.34", 0, true},
	}

	for _, tt := range tests {
		result, err := parseValue(tt.input, TypeInt64, ",")
		if tt.hasError {
			if err == nil {
				t.Errorf("parseValue(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseValue(%s) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseValue(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	}
}

func TestParseValueFloat64(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"123.45", 123.45, false},
		{"-456.78", -456.78, false},
		{"0", 0, false},
		{"1e10", 1e10, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := parseValue(tt.input, TypeFloat64, ",")
		if tt.hasError {
			if err == nil {
				t.Errorf("parseValue(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseValue(%s) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseValue(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	}
}

func TestParseValueBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		hasError bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"1", true, false},
		{"0", false, false},
		{"TRUE", true, false},
		{"FALSE", false, false},
		{"invalid", false, true},
		{"yes", false, true}, // Go's ParseBool doesn't support yes/no
	}

	for _, tt := range tests {
		result, err := parseValue(tt.input, TypeBool, ",")
		if tt.hasError {
			if err == nil {
				t.Errorf("parseValue(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseValue(%s) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseValue(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	}
}

func TestParseValueDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"1h", time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"15s", 15 * time.Second, false},
		{"1h30m", 90 * time.Minute, false},
		{"7d", 7 * 24 * time.Hour, false}, // day format
		{"1d", 24 * time.Hour, false},     // day format
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := parseValue(tt.input, TypeDuration, ",")
		if tt.hasError {
			if err == nil {
				t.Errorf("parseValue(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseValue(%s) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseValue(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	}
}

func TestParseValueStringSlice(t *testing.T) {
	tests := []struct {
		input     string
		delimiter string
		expected  []string
		isNil     bool
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}, false},
		{"a, b, c", ",", []string{"a", "b", "c"}, false}, // with spaces
		{"a|b|c", "|", []string{"a", "b", "c"}, false},
		{"single", ",", []string{"single"}, false},
		{"", ",", nil, true},                     // empty string returns nil
		{"a,,b", ",", []string{"a", "b"}, false}, // empty elements removed
	}

	for _, tt := range tests {
		result, err := parseValue(tt.input, TypeStringSlice, tt.delimiter)
		if err != nil {
			t.Errorf("parseValue(%s) unexpected error: %v", tt.input, err)
			continue
		}
		if tt.isNil {
			if result != nil {
				t.Errorf("parseValue(%s) expected nil, got %v", tt.input, result)
			}
			continue
		}
		slice, ok := result.([]string)
		if !ok {
			t.Errorf("parseValue(%s) result is not []string, got %T", tt.input, result)
			continue
		}
		if len(slice) != len(tt.expected) {
			t.Errorf("parseValue(%s) = %v, expected %v", tt.input, slice, tt.expected)
			continue
		}
		for i, v := range slice {
			if v != tt.expected[i] {
				t.Errorf("parseValue(%s)[%d] = %s, expected %s", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestParseValueInt64Slice(t *testing.T) {
	tests := []struct {
		input     string
		delimiter string
		expected  []int64
		hasError  bool
		isNil     bool
	}{
		{"1,2,3", ",", []int64{1, 2, 3}, false, false},
		{"1, 2, 3", ",", []int64{1, 2, 3}, false, false}, // with spaces
		{"1|2|3", "|", []int64{1, 2, 3}, false, false},
		{"42", ",", []int64{42}, false, false},
		{"", ",", nil, false, true},                // empty string returns nil
		{"1,,2", ",", []int64{1, 2}, false, false}, // empty elements removed
		{"1,invalid,3", ",", nil, true, false},
	}

	for _, tt := range tests {
		result, err := parseValue(tt.input, TypeInt64Slice, tt.delimiter)
		if tt.hasError {
			if err == nil {
				t.Errorf("parseValue(%s) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseValue(%s) unexpected error: %v", tt.input, err)
			continue
		}
		if tt.isNil {
			if result != nil {
				t.Errorf("parseValue(%s) expected nil, got %v", tt.input, result)
			}
			continue
		}
		slice, ok := result.([]int64)
		if !ok {
			t.Errorf("parseValue(%s) result is not []int64, got %T", tt.input, result)
			continue
		}
		if len(slice) != len(tt.expected) {
			t.Errorf("parseValue(%s) = %v, expected %v", tt.input, slice, tt.expected)
			continue
		}
		for i, v := range slice {
			if v != tt.expected[i] {
				t.Errorf("parseValue(%s)[%d] = %d, expected %d", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestParseValueUnknownType(t *testing.T) {
	_, err := parseValue("test", ValueType(99), ",")
	if err == nil {
		t.Error("parseValue with unknown type should return error")
	}
}

func TestSourcePriorityTypes(t *testing.T) {
	// Test SourceType String() method
	tests := []struct {
		source   SourceType
		expected string
	}{
		{SourceDefault, "default"},
		{SourceFlag, "flag"},
		{SourceEnv, "environment"},
		{SourceFile, "file"},
		{SourceType(999), "unknown"},
	}

	for _, test := range tests {
		if test.source.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.source.String())
		}
	}
}

func TestPriorityPresets(t *testing.T) {
	// Test that presets are correctly defined
	if len(PriorityFlagEnvDefault) != 3 {
		t.Error("PriorityFlagEnvDefault should have 3 elements")
	}
	if PriorityFlagEnvDefault[0] != SourceFlag || PriorityFlagEnvDefault[1] != SourceEnv || PriorityFlagEnvDefault[2] != SourceDefault {
		t.Error("PriorityFlagEnvDefault should be [Flag, Env, Default]")
	}

	if len(PriorityEnvFlagDefault) != 3 {
		t.Error("PriorityEnvFlagDefault should have 3 elements")
	}
	if PriorityEnvFlagDefault[0] != SourceEnv || PriorityEnvFlagDefault[1] != SourceFlag || PriorityEnvFlagDefault[2] != SourceDefault {
		t.Error("PriorityEnvFlagDefault should be [Env, Flag, Default]")
	}

	if len(PriorityFileEnvFlagDefault) != 4 {
		t.Error("PriorityFileEnvFlagDefault should have 4 elements")
	}
	if PriorityFileEnvFlagDefault[0] != SourceFile || PriorityFileEnvFlagDefault[1] != SourceEnv || PriorityFileEnvFlagDefault[2] != SourceFlag || PriorityFileEnvFlagDefault[3] != SourceDefault {
		t.Error("PriorityFileEnvFlagDefault should be [File, Env, Flag, Default]")
	}
}
