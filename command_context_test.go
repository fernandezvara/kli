// cli/command_context_test.go
package kli

import (
	"testing"
)

// TestCommandContextHelpers tests CommandContext helper methods
func TestCommandContextHelpers(t *testing.T) {
	ctx := NewCommandContext([]string{}, New(), "test", "")

	// Test Set and Get
	ctx.Set("key", "value")
	if val, exists := ctx.GetData("key"); !exists {
		t.Error("Key not found after Set")
	} else if val != "value" {
		t.Errorf("Expected 'value', got %v", val)
	}

	// Test GetData string
	ctx.Set("str_key", "string_value")
	if value, exists := ctx.GetData("str_key"); exists {
		if str, ok := value.(string); ok && str == "string_value" {
			// Good
		} else {
			t.Errorf("Expected 'string_value', got %v", value)
		}
	} else {
		t.Error("str_key should exist")
	}

	// Test GetData int
	ctx.Set("int_key", 42)
	if value, exists := ctx.GetData("int_key"); exists {
		if i, ok := value.(int); ok && i == 42 {
			// Good
		} else {
			t.Errorf("Expected 42, got %v", value)
		}
	} else {
		t.Error("int_key should exist")
	}

	// Test GetData bool
	ctx.Set("bool_key", true)
	if value, exists := ctx.GetData("bool_key"); exists {
		if b, ok := value.(bool); ok && b {
			// Good
		} else {
			t.Error("Expected true, got false")
		}
	} else {
		t.Error("bool_key should exist")
	}

	// Test non-existent keys
	if _, exists := ctx.GetData("nonexistent"); exists {
		t.Error("Non-existent key should not exist")
	}
	if value, exists := ctx.GetData("nonexistent"); exists {
		if str, ok := value.(string); ok && str != "" {
			t.Errorf("Expected empty string for non-existent key, got %s", str)
		}
	}
	if value, exists := ctx.GetData("nonexistent"); exists {
		if i, ok := value.(int); ok && i != 0 {
			t.Errorf("Expected 0 for non-existent key, got %d", i)
		}
	}
	if value, exists := ctx.GetData("nonexistent"); exists {
		if b, ok := value.(bool); ok && b {
			t.Error("Expected false for non-existent key, got true")
		}
	}
}

func TestCommandContextWithExecution(t *testing.T) {
	cfg := New()
	cfg.Define("TEST_VALUE").String().Default("test")

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test that execution context is initialized
	if ctx.execution == nil {
		t.Error("Expected execution context to be initialized")
	}

	if ctx.execution.GetCommand() != "test" {
		t.Errorf("Expected command 'test', got '%s'", ctx.execution.GetCommand())
	}
}
