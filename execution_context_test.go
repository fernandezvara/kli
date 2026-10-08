package kli

import (
	"testing"
)

func TestExecutionContext(t *testing.T) {
	// Test basic execution context creation
	ctx := NewExecutionContext("test-command")

	if ctx.GetCommand() != "test-command" {
		t.Errorf("Expected command 'test-command', got '%s'", ctx.GetCommand())
	}

	// Test error collection
	if ctx.HasErrors() {
		t.Error("Expected no errors initially")
	}

	// Test collecting an error
	ctx.CollectError(nil, "test-key", "string", "", "test error", false)

	if !ctx.HasErrors() {
		t.Error("Expected errors after collecting one")
	}

	errors := ctx.GetErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	if errors[0].Key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", errors[0].Key)
	}

	// Test clearing errors
	ctx.Clear()

	if ctx.HasErrors() {
		t.Error("Expected no errors after clearing")
	}
}

// TestGetErrorCollectionIntegration tests the complete error collection flow
func TestGetErrorCollectionIntegration(t *testing.T) {
	cfg := New()

	// Define required configuration with flags and env vars
	cfg.Define("PORT").Int64().Flag("port").Env("PORT").Required()
	cfg.Define("HOST").String().Flag("host").Env("HOST").Required()
	cfg.Define("API_KEY").String().Flag("api-key").Env("API_KEY").Secret().Required()

	_ = cfg.Execute([]string{"test"})

	// Clear any previous errors
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Execute - this should collect errors and exit
	// We can't test the os.Exit directly, but we can verify error collection
	// by calling the Get functions directly
	// Note: Get functions now return (T, error) for missing data
	_, err := Get[int64](ctx, "PORT")
	if err == nil {
		t.Errorf("Expected error for missing PORT, got nil")
	}
	_, err = Get[string](ctx, "HOST")
	if err == nil {
		t.Errorf("Expected error for missing HOST, got nil")
	}

	_, err = Get[string](ctx, "API_KEY")
	if err == nil {
		t.Errorf("Expected error for missing API_KEY, got nil")
	}

	// Note: Required keys don't collect errors, they return errors directly
	// The new behavior separates required validation from error collection
	// Only non-required keys collect errors in the execution context

	// Test non-required data still collects errors
	_, err = Get[string](ctx, "NONEXISTENT_KEY")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	// Now we should have collected errors for the non-required key
	if !ctx.execution.HasErrors() {
		t.Error("Expected errors to be collected for non-required data")
	}

	collected := ctx.execution.GetErrors()
	if len(collected) == 0 {
		t.Errorf("Expected collected errors for non-required data, got %d", len(collected))
	}

	// Test that we can add more errors
	ctx.execution.CollectError(cfg, "ANOTHER_KEY", "not found", "", "test error", false)
	collectedAfter := ctx.execution.GetErrors()
	if len(collectedAfter) != len(collected)+1 {
		t.Errorf("Expected %d collected errors, got %d", len(collected)+1, len(collectedAfter))
	}
}
