package kli

import (
	"testing"
)

func TestMustGet(t *testing.T) {
	cfg := New()

	cfg.Define("PORT").Int64().Default(8080)
	cfg.Define("HOST").String() // No default, not required

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Unexpected errors: %v", err)
	}

	// Create context for new API
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test MustGet with existing value
	port := MustGet[int64](ctx, "PORT")
	if port != 8080 {
		t.Errorf("MustGet should return 8080, got %d", port)
	}

	// Test MustGet with missing key (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustGet to panic for missing key")
		}
	}()

	MustGet[string](ctx, "MISSING_KEY")
}

func TestGetWithMissingKey(t *testing.T) {
	cfg := New()

	cfg.Define("MISSING_KEY").String()

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Unexpected errors: %v", err)
	}

	// Create context for new API
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test Get with missing key
	_, err := Get[string](ctx, "MISSING_KEY")
	if err == nil {
		t.Error("Expected error for missing key")
	}

	// Verify error was collected
	if !ctx.execution.HasErrors() {
		t.Error("Expected error to be collected for missing key")
	}

	collected := ctx.execution.GetErrors()
	if len(collected) == 0 {
		t.Error("Expected error to be collected for missing key")
	}

	if collected[0].Key != "MISSING_KEY" {
		t.Errorf("Expected key 'MISSING_KEY', got '%s'", collected[0].Key)
	}
}

func TestGetWithTypeConversion(t *testing.T) {
	cfg := New()

	cfg.Define("PORT").String().Default("8080") // String should convert to int64

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Unexpected errors: %v", err)
	}

	// Create context for new API
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test Get with type conversion (should now work)
	value, err := Get[int64](ctx, "PORT")
	if err != nil {
		t.Errorf("Expected successful conversion, got error: %v", err)
	}

	// Verify the converted value
	if value != 8080 {
		t.Errorf("Expected value 8080, got %d", value)
	}

	// Verify no errors were collected (conversion succeeded)
	if ctx.execution.HasErrors() {
		t.Error("Expected no errors for successful conversion")
	}
}

func TestGetWithSecret(t *testing.T) {
	cfg := New()

	cfg.Define("API_KEY").String().Secret()

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Unexpected errors: %v", err)
	}

	// Create context for new API
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test Get with secret (should return error)
	_, err := Get[string](ctx, "API_KEY")
	if err == nil {
		t.Error("Expected error for secret access")
	}

	if err.Error() != "validation error: configuration 'API_KEY' is secret, use GetSecret() instead" {
		t.Errorf("Expected secret access error, got: %v", err)
	}

	// Verify error was collected
	if !ctx.execution.HasErrors() {
		t.Error("Expected error to be collected for secret access")
	}

	collected := ctx.execution.GetErrors()
	if len(collected) == 0 {
		t.Error("Expected error to be collected for secret access")
	}

	if collected[0].Key != "API_KEY" {
		t.Errorf("Expected key 'API_KEY', got '%s'", collected[0].Key)
	}

	if !collected[0].IsSecret {
		t.Error("Expected error to be marked as secret")
	}
}

func TestGetErrorDisplayName(t *testing.T) {
	cfg := New()

	cfg.Define("PORT").Int64().Flag("port").Env("PORT")
	cfg.Define("DATABASE_URL").String().Env("DATABASE_URL").Secret()
	cfg.Define("DEBUG").Bool().Env("DEBUG")

	// Don't process config since we're just testing display name formatting
	// Create context for new API
	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Collect some errors to test display name formatting
	ctx.execution.CollectError(cfg, "PORT", "not found", "", "key not defined", false)
	ctx.execution.CollectError(cfg, "DATABASE_URL", "secret", "", "use GetSecret() instead", true)
	ctx.execution.CollectError(cfg, "DEBUG", "not found", "", "key not defined", false)

	collected := ctx.execution.GetErrors()
	if len(collected) != 3 {
		t.Errorf("Expected 3 collected errors, got %d", len(collected))
	}

	// Test display name formatting
	portDisplayName := getErrorDisplayName(collected[0], cfg)
	if portDisplayName != "--port int64 (env: PORT)" {
		t.Errorf("Expected '--port int64 (env: PORT)', got '%s'", portDisplayName)
	}

	dbDisplayName := getErrorDisplayName(collected[1], cfg)
	if dbDisplayName != "DATABASE_URL string (secret)" {
		t.Errorf("Expected 'DATABASE_URL string (secret)', got '%s'", dbDisplayName)
	}

	debugDisplayName := getErrorDisplayName(collected[2], cfg)
	if debugDisplayName != "DEBUG bool" {
		t.Errorf("Expected 'DEBUG bool', got '%s'", debugDisplayName)
	}
}

func TestDisplayGetErrors(t *testing.T) {
	cfg := New()
	cfg.Define("DATABASE_URL").String().Env("DATABASE_URL").Required().Secret().Description("Database connection string")
	cfg.Define("PORT").Int64().Flag("port").Range(1, 65535).Description("HTTP server port")

	ctx := NewCommandContext([]string{}, cfg, "start", "")

	ctx.execution.CollectError(cfg, "DATABASE_URL", "not found", "", "key not defined", false)
	ctx.execution.CollectError(cfg, "PORT", "validation", "", "value 99999 is greater than maximum 65535", false)

	outputStr, err := ctx.execution.renderErrorsWithCommand(nil, nil)
	if err != nil {
		t.Fatalf("renderErrorsWithCommand returned error: %v", err)
	}

	if !contains(outputStr, "start [options]") {
		t.Error("Expected templated output to contain command usage")
	}
	if !contains(outputStr, "Configuration errors:") {
		t.Error("Expected templated output to contain configuration errors section")
	}
	if !contains(outputStr, "DATABASE_URL string (required, secret) -> key not defined") {
		t.Error("Expected templated output to contain DATABASE_URL error")
	}
	if !contains(outputStr, "--port int64") && !contains(outputStr, "value 99999 is greater than maximum 65535") {
		t.Error("Expected templated output to contain PORT validation error")
	}
}

func TestGetWithErrorCollection(t *testing.T) {
	cfg := New()
	cfg.Define("MISSING_KEY").String().Required()

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test that Get returns error for missing key
	_, err := Get[string](ctx, "MISSING_KEY")
	if err == nil {
		t.Error("Expected error for missing required key")
	}

	// Note: Required keys don't collect errors, they just return errors directly
	// The warning is logged but not collected in execution context
	// This is the new behavior - required data validation is separate from error collection
}

func TestMustGetWithExecutionContext(t *testing.T) {
	cfg := New()
	cfg.Define("TEST_VALUE").String().Default("test")

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Configuration errors: %v", err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test MustGet with existing key
	value := MustGet[string](ctx, "TEST_VALUE")
	if value != "test" {
		t.Errorf("Expected 'test', got '%s'", value)
	}

	// Test MustGet with missing key (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustGet to panic for missing key")
		}
	}()

	MustGet[string](ctx, "MISSING_KEY")
}

// TestAPIChanges verifies that our API surface reduction works correctly
func TestAPIChanges(t *testing.T) {
	// Test that new Get[T] API works
	cfg := New()
	cfg.Define("PORT").Int64().Default(8080)

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Config processing failed: %v", err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test new Get[T] signature
	port, err := Get[int64](ctx, "PORT")
	if err != nil {
		t.Errorf("Get[T] failed: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected 8080, got %d", port)
	}

	// Test getting non-existent key returns error
	_, err = Get[bool](ctx, "DEBUG")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

// TestGetPerformance tests the performance optimizations in Get[T]
func TestGetPerformance(t *testing.T) {
	cfg := New()
	cfg.Define("PORT").Int64().Default(8080)
	cfg.Define("HOST").String().Default("localhost")
	cfg.Define("DEBUG").Bool().Default(false)
	cfg.Define("RATE").Float64().Default(0.5)

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Failed to process config: %v", err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test multiple Get calls to verify type caching
	for range 100 {
		// Test int64
		port, err := Get[int64](ctx, "PORT")
		if err != nil {
			t.Errorf("Failed to get PORT: %v", err)
		}
		if port != 8080 {
			t.Errorf("Expected PORT=8080, got %d", port)
		}

		// Test string
		host, err := Get[string](ctx, "HOST")
		if err != nil {
			t.Errorf("Failed to get HOST: %v", err)
		}
		if host != "localhost" {
			t.Errorf("Expected HOST=localhost, got %s", host)
		}

		// Test bool
		debug, err := Get[bool](ctx, "DEBUG")
		if err != nil {
			t.Errorf("Failed to get DEBUG: %v", err)
		}
		if debug != false {
			t.Errorf("Expected DEBUG=false, got %v", debug)
		}

		// Test float64
		rate, err := Get[float64](ctx, "RATE")
		if err != nil {
			t.Errorf("Failed to get RATE: %v", err)
		}
		if rate != 0.5 {
			t.Errorf("Expected RATE=0.5, got %f", rate)
		}
	}
}

func TestSliceTypeNoPanic(t *testing.T) {
	// Test that slice types don't cause panic in typeDescription
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("typeDescription panicked with slice types: %v", r)
		}
	}()

	// Test various slice types
	sliceTests := []struct {
		name     string
		value    any
		expected string
	}{
		{"string slice", []string{"a", "b"}, "[]string"},
		{"int64 slice", []int64{1, 2}, "[]int64"},
		{"int slice", []int{1, 2}, "[]int"},
		{"empty string slice", []string{}, "[]string"},
		{"empty int64 slice", []int64{}, "[]int64"},
		{"empty int slice", []int{}, "[]int"},
	}

	for _, tt := range sliceTests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeDescription(tt.value)
			if result != tt.expected {
				t.Errorf("typeDescription(%v) = %q, expected %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestInt64Consistency(t *testing.T) {
	// Test that int64 always returns "int64" never "int"
	int64Value := int64(123)
	result := typeDescription(int64Value)
	if result != "int64" {
		t.Errorf("typeDescription(int64) = %q, expected \"int64\"", result)
	}

	// Test that int returns "int"
	intValue := int(123)
	result = typeDescription(intValue)
	if result != "int" {
		t.Errorf("typeDescription(int) = %q, expected \"int\"", result)
	}
}

func TestTypeCachingWorks(t *testing.T) {
	// Test that type caching still works for basic types
	basicTests := []struct {
		name     string
		value    any
		expected string
	}{
		{"string", "test", "string"},
		{"int64", int64(123), "int64"},
		{"int", int(123), "int"},
		{"bool", true, "bool"},
		{"float64", 3.14, "float64"},
	}

	for _, tt := range basicTests {
		t.Run(tt.name, func(t *testing.T) {
			// Call multiple times to test caching
			result1 := typeDescription(tt.value)
			result2 := typeDescription(tt.value)

			if result1 != tt.expected {
				t.Errorf("First call typeDescription(%v) = %q, expected %q", tt.value, result1, tt.expected)
			}
			if result2 != tt.expected {
				t.Errorf("Second call typeDescription(%v) = %q, expected %q", tt.value, result2, tt.expected)
			}
			if result1 != result2 {
				t.Errorf("Cached results differ: first=%q, second=%q", result1, result2)
			}
		})
	}
}

func TestSliceTypeCachingWorks(t *testing.T) {
	// Test that type caching works for slice types
	sliceValue := []string{"a", "b"}

	// Call multiple times to test caching
	result1 := typeDescription(sliceValue)
	result2 := typeDescription(sliceValue)

	expected := "[]string"
	if result1 != expected {
		t.Errorf("First call typeDescription(%v) = %q, expected %q", sliceValue, result1, expected)
	}
	if result2 != expected {
		t.Errorf("Second call typeDescription(%v) = %q, expected %q", sliceValue, result2, expected)
	}
	if result1 != result2 {
		t.Errorf("Cached slice results differ: first=%q, second=%q", result1, result2)
	}
}

// TestSliceTypeRetrievalFixed tests that the slice regression is fixed
func TestSliceTypeRetrievalFixed(t *testing.T) {
	cfg := New()

	cfg.Define("TAGS").
		StringSlice().
		Default([]string{"ssh-rsa", "ssh-ed25519"})

	cfg.Define("NUMBERS").
		Int64Slice().
		Default([]int64{1, 2, 3})

	cfg.Define("PORTS").
		Int64Slice().
		Default([]int64{8080, 8081})

	// Test configuration processing
	err := cfg.Execute([]string{"test"})
	if err != nil {
		t.Fatalf("Config execution failed: %v", err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Test Get[[]string] - should work without workaround
	tags, err := Get[[]string](ctx, "TAGS")
	if err != nil {
		t.Fatalf("Get[[]string] failed: %v", err)
	}
	expectedTags := []string{"ssh-rsa", "ssh-ed25519"}
	if len(tags) != len(expectedTags) {
		t.Fatalf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}
	for i, tag := range expectedTags {
		if tags[i] != tag {
			t.Errorf("Expected tag[%d] = %q, got %q", i, tag, tags[i])
		}
	}

	// Test Get[[]int64]
	numbers, err := Get[[]int64](ctx, "NUMBERS")
	if err != nil {
		t.Fatalf("Get[[]int64] failed: %v", err)
	}
	expectedNumbers := []int64{1, 2, 3}
	if len(numbers) != len(expectedNumbers) {
		t.Fatalf("Expected %d numbers, got %d", len(expectedNumbers), len(numbers))
	}
	for i, num := range expectedNumbers {
		if numbers[i] != num {
			t.Errorf("Expected number[%d] = %d, got %d", i, num, numbers[i])
		}
	}

	// Test Get[[]int64] for ports
	ports, err := Get[[]int64](ctx, "PORTS")
	if err != nil {
		t.Fatalf("Get[[]int64] failed: %v", err)
	}
	expectedPorts := []int64{8080, 8081}
	if len(ports) != len(expectedPorts) {
		t.Fatalf("Expected %d ports, got %d", len(expectedPorts), len(ports))
	}
	for i, port := range expectedPorts {
		if ports[i] != port {
			t.Errorf("Expected port[%d] = %d, got %d", i, port, ports[i])
		}
	}
}

// TestNoSliceWorkaroundNeeded ensures users don't need workarounds
func TestNoSliceWorkaroundNeeded(t *testing.T) {
	cfg := New()

	cfg.Define("ALGORITHMS").
		StringSlice().
		Default([]string{"ssh-rsa", "ssh-ed25519"})

	err := cfg.Execute([]string{"test"})
	if err != nil {
		t.Fatalf("Config execution failed: %v", err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Should work directly without string splitting workaround
	algorithms, err := Get[[]string](ctx, "ALGORITHMS")
	if err != nil {
		t.Fatalf("Get[[]string] failed: %v", err)
	}

	expected := []string{"ssh-rsa", "ssh-ed25519"}
	if len(algorithms) != len(expected) {
		t.Fatalf("Expected %d algorithms, got %d", len(expected), len(algorithms))
	}
	for i, algo := range expected {
		if algorithms[i] != algo {
			t.Errorf("Expected algorithm[%d] = %q, got %q", i, algo, algorithms[i])
		}
	}

	// Should NOT return string representation when requesting string
	_, err = Get[string](ctx, "ALGORITHMS")
	if err == nil {
		t.Error("Expected type mismatch error when getting string instead of []string")
	}
}

// TestAllSourcesSliceConsistency tests slice consistency across all value sources
func TestAllSourcesSliceConsistency(t *testing.T) {
	// Test each source type consistently
	testCases := []struct {
		name     string
		args     []string
		expected []string
		env      string
	}{
		{"default", []string{}, []string{"default1", "default2"}, ""},
		{"env", []string{}, []string{"env1", "env2", "env3"}, "env1,env2,env3"},
		{"flag", []string{"--tags", "flag1,flag2"}, []string{"flag1", "flag2"}, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable for this test
			if tc.env != "" {
				t.Setenv("TEST_TAGS", tc.env)
			}

			// Create fresh config for each test
			testCfg := New()
			testCfg.Define("TAGS").
				StringSlice().
				Env("TEST_TAGS").
				Flag("tags").
				Default([]string{"default1", "default2"})

			err := testCfg.Execute(append([]string{"test"}, tc.args...))
			if err != nil {
				t.Fatalf("Config execution failed: %v", err)
			}

			ctx := NewCommandContext([]string{}, testCfg, "test", "")
			tags, err := Get[[]string](ctx, "TAGS")
			if err != nil {
				t.Fatalf("Get[[]string] failed: %v", err)
			}

			if len(tags) != len(tc.expected) {
				t.Fatalf("Expected %d tags, got %d", len(tc.expected), len(tags))
			}
			for i, tag := range tc.expected {
				if tags[i] != tag {
					t.Errorf("Expected tag[%d] = %q, got %q", i, tag, tags[i])
				}
			}
		})
	}
}

// TestSecretSecurityViolation tests that Get[T] properly blocks secret access
func TestSecretSecurityViolation(t *testing.T) {
	cfg := New()
	cfg.Define("SECRET_KEY").String().Secret().Default("secret-value")
	cfg.Define("NORMAL_KEY").String().Default("normal-value")

	// Set environment variable for secret (optional, will use default)
	t.Setenv("SECRET_KEY", "secret-value")
	t.Setenv("NORMAL_KEY", "env-value")

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Config processing failed: %v", err)
	}

	// Create command context
	ctx := &CommandContext{
		GlobalConfig: cfg,
		execution:    NewExecutionContext("test"),
	}

	// Try to access secret with Get[T] - should fail
	_, err := Get[string](ctx, "SECRET_KEY")
	if err == nil {
		t.Error("Get[string] should fail for secret keys")
	}

	if err.Error() != "validation error: configuration 'SECRET_KEY' is secret, use GetSecret() instead" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	// Access secret properly with GetSecret
	secret := cfg.GetSecret("SECRET_KEY")
	if !secret.IsSet() || secret.String() != "secret-value" {
		t.Error("GetSecret should work for secret keys")
	}

	// Normal key should work with Get[T]
	normalValue, err := Get[string](ctx, "NORMAL_KEY")
	if err != nil {
		t.Errorf("Get[string] should work for normal keys: %v", err)
	}

	if normalValue != "normal-value" {
		t.Errorf("Expected 'normal-value', got '%s'", normalValue)
	}

	// Test Has method behavior
	if cfg.Has("SECRET_KEY") {
		t.Error("Has should return false for secret keys")
	}

	if !cfg.HasSecret("SECRET_KEY") {
		t.Error("HasSecret should return true for secret keys")
	}

	if !cfg.Has("NORMAL_KEY") {
		t.Error("Has should return true for normal keys")
	}

	cfg.Destroy()
}
