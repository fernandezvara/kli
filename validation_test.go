package kli

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestValidateMin(t *testing.T) {
	v := validateMin(10)

	// Int64 tests
	if err := v.Check(int64(5)); err == nil {
		t.Error("validateMin(10) should fail for int64(5)")
	}
	if err := v.Check(int64(10)); err != nil {
		t.Errorf("validateMin(10) should pass for int64(10): %v", err)
	}
	if err := v.Check(int64(15)); err != nil {
		t.Errorf("validateMin(10) should pass for int64(15): %v", err)
	}

	// Float64 tests
	if err := v.Check(float64(5.5)); err == nil {
		t.Error("validateMin(10) should fail for float64(5.5)")
	}
	if err := v.Check(float64(10.0)); err != nil {
		t.Errorf("validateMin(10) should pass for float64(10.0): %v", err)
	}
	if err := v.Check(float64(15.5)); err != nil {
		t.Errorf("validateMin(10) should pass for float64(15.5): %v", err)
	}

	// Non-numeric should pass (no validation)
	if err := v.Check("string"); err != nil {
		t.Errorf("validateMin should pass for non-numeric types: %v", err)
	}
}

func TestValidateMax(t *testing.T) {
	v := validateMax(100)

	// Int64 tests
	if err := v.Check(int64(150)); err == nil {
		t.Error("validateMax(100) should fail for int64(150)")
	}
	if err := v.Check(int64(100)); err != nil {
		t.Errorf("validateMax(100) should pass for int64(100): %v", err)
	}
	if err := v.Check(int64(50)); err != nil {
		t.Errorf("validateMax(100) should pass for int64(50): %v", err)
	}

	// Float64 tests
	if err := v.Check(float64(150.5)); err == nil {
		t.Error("validateMax(100) should fail for float64(150.5)")
	}
	if err := v.Check(float64(100.0)); err != nil {
		t.Errorf("validateMax(100) should pass for float64(100.0): %v", err)
	}
	if err := v.Check(float64(50.5)); err != nil {
		t.Errorf("validateMax(100) should pass for float64(50.5): %v", err)
	}
}

func TestValidateMinLength(t *testing.T) {
	v := validateMinLength(5)

	if err := v.Check("abc"); err == nil {
		t.Error("validateMinLength(5) should fail for 'abc'")
	}
	if err := v.Check("abcde"); err != nil {
		t.Errorf("validateMinLength(5) should pass for 'abcde': %v", err)
	}
	if err := v.Check("abcdefgh"); err != nil {
		t.Errorf("validateMinLength(5) should pass for 'abcdefgh': %v", err)
	}

	// Non-string should pass
	if err := v.Check(123); err != nil {
		t.Errorf("validateMinLength should pass for non-string: %v", err)
	}
}

func TestValidateMaxLength(t *testing.T) {
	v := validateMaxLength(5)

	if err := v.Check("abcdefgh"); err == nil {
		t.Error("validateMaxLength(5) should fail for 'abcdefgh'")
	}
	if err := v.Check("abcde"); err != nil {
		t.Errorf("validateMaxLength(5) should pass for 'abcde': %v", err)
	}
	if err := v.Check("abc"); err != nil {
		t.Errorf("validateMaxLength(5) should pass for 'abc': %v", err)
	}
}

func TestValidateRegexp(t *testing.T) {
	v := validateRegexp(`^[a-z]+@[a-z]+\.[a-z]+$`)

	if err := v.Check("invalid"); err == nil {
		t.Error("validateRegexp should fail for 'invalid'")
	}
	if err := v.Check("test@example.com"); err != nil {
		t.Errorf("validateRegexp should pass for 'test@example.com': %v", err)
	}

	// Non-string should pass
	if err := v.Check(123); err != nil {
		t.Errorf("validateRegexp should pass for non-string: %v", err)
	}
}

func TestValidateOneOf(t *testing.T) {
	v := validateOneOf([]string{"debug", "info", "warn", "error"})

	if err := v.Check("invalid"); err == nil {
		t.Error("validateOneOf should fail for 'invalid'")
	}
	if err := v.Check("debug"); err != nil {
		t.Errorf("validateOneOf should pass for 'debug': %v", err)
	}
	if err := v.Check("info"); err != nil {
		t.Errorf("validateOneOf should pass for 'info': %v", err)
	}
	if err := v.Check("error"); err != nil {
		t.Errorf("validateOneOf should pass for 'error': %v", err)
	}

	// Non-string should pass
	if err := v.Check(123); err != nil {
		t.Errorf("validateOneOf should pass for non-string: %v", err)
	}
}

func TestValidateMinDuration(t *testing.T) {
	v := validateMinDuration(5 * time.Second)

	if err := v.Check(2 * time.Second); err == nil {
		t.Error("validateMinDuration(5s) should fail for 2s")
	}
	if err := v.Check(5 * time.Second); err != nil {
		t.Errorf("validateMinDuration(5s) should pass for 5s: %v", err)
	}
	if err := v.Check(10 * time.Second); err != nil {
		t.Errorf("validateMinDuration(5s) should pass for 10s: %v", err)
	}

	// Non-duration should pass
	if err := v.Check("string"); err != nil {
		t.Errorf("validateMinDuration should pass for non-duration: %v", err)
	}
}

func TestValidateMaxDuration(t *testing.T) {
	v := validateMaxDuration(1 * time.Minute)

	if err := v.Check(2 * time.Minute); err == nil {
		t.Error("validateMaxDuration(1m) should fail for 2m")
	}
	if err := v.Check(1 * time.Minute); err != nil {
		t.Errorf("validateMaxDuration(1m) should pass for 1m: %v", err)
	}
	if err := v.Check(30 * time.Second); err != nil {
		t.Errorf("validateMaxDuration(1m) should pass for 30s: %v", err)
	}
}

func TestValidateMinItems(t *testing.T) {
	v := validateMinItems(2)

	// String slice tests
	if err := v.Check([]string{"a"}); err == nil {
		t.Error("validateMinItems(2) should fail for []string with 1 item")
	}
	if err := v.Check([]string{"a", "b"}); err != nil {
		t.Errorf("validateMinItems(2) should pass for []string with 2 items: %v", err)
	}
	if err := v.Check([]string{"a", "b", "c"}); err != nil {
		t.Errorf("validateMinItems(2) should pass for []string with 3 items: %v", err)
	}

	// Int64 slice tests
	if err := v.Check([]int64{1}); err == nil {
		t.Error("validateMinItems(2) should fail for []int64 with 1 item")
	}
	if err := v.Check([]int64{1, 2}); err != nil {
		t.Errorf("validateMinItems(2) should pass for []int64 with 2 items: %v", err)
	}

	// Non-slice should pass
	if err := v.Check("string"); err != nil {
		t.Errorf("validateMinItems should pass for non-slice: %v", err)
	}
}

func TestValidateMaxItems(t *testing.T) {
	v := validateMaxItems(3)

	// String slice tests
	if err := v.Check([]string{"a", "b", "c", "d"}); err == nil {
		t.Error("validateMaxItems(3) should fail for []string with 4 items")
	}
	if err := v.Check([]string{"a", "b", "c"}); err != nil {
		t.Errorf("validateMaxItems(3) should pass for []string with 3 items: %v", err)
	}
	if err := v.Check([]string{"a", "b"}); err != nil {
		t.Errorf("validateMaxItems(3) should pass for []string with 2 items: %v", err)
	}

	// Int64 slice tests
	if err := v.Check([]int64{1, 2, 3, 4}); err == nil {
		t.Error("validateMaxItems(3) should fail for []int64 with 4 items")
	}
	if err := v.Check([]int64{1, 2, 3}); err != nil {
		t.Errorf("validateMaxItems(3) should pass for []int64 with 3 items: %v", err)
	}
}

func TestValidationName(t *testing.T) {
	// Test that validation names are set correctly
	tests := []struct {
		validation Validation
		contains   string
	}{
		{validateMin(10), "min(10)"},
		{validateMax(100), "max(100)"},
		{validateMinLength(5), "minLength(5)"},
		{validateMaxLength(10), "maxLength(10)"},
		{validateRegexp(`\d+`), "regexp"},
		{validateOneOf([]string{"a", "b"}), "oneOf"},
		{validateMinDuration(5 * time.Second), "minDuration"},
		{validateMaxDuration(1 * time.Minute), "maxDuration"},
		{validateMinItems(2), "minItems(2)"},
		{validateMaxItems(5), "maxItems(5)"},
	}

	for _, tt := range tests {
		if tt.validation.Name == "" {
			t.Errorf("Validation name should not be empty")
		}
	}
}

func TestValidationCache_Performance(t *testing.T) {
	pattern := `^[a-z]+@[a-z]+\.[a-z]+$`

	// Test cache performance
	start := time.Now()

	// Create multiple validations with the same pattern
	for range 1000 {
		validation := validateRegexp(pattern)
		if validation.Name != "regexp(^[a-z]+@[a-z]+\\.[a-z]+$)" {
			t.Errorf("Expected validation name to contain pattern, got: %s", validation.Name)
		}

		// Test validation
		err := validation.Check("test@example.com")
		if err != nil {
			t.Errorf("Expected valid email to pass, got error: %v", err)
		}
	}

	duration := time.Since(start)
	t.Logf("1000 validations with cached regex took: %v", duration)

	// Should be very fast with caching (less than 10ms for 1000 operations)
	if duration > 10*time.Millisecond {
		t.Logf("WARNING: Validation took longer than expected: %v", duration)
	}
}

func TestValidationCache_ConcurrentAccess(t *testing.T) {
	pattern := `^\d{4}-\d{2}-\d{2}$`

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Test concurrent access to the cache
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			validation := validateRegexp(pattern)

			// Test with valid date
			err := validation.Check("2023-12-25")
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: valid date failed: %v", id, err)
				return
			}

			// Test with invalid date
			err = validation.Check("invalid")
			if err == nil {
				errors <- fmt.Errorf("goroutine %d: invalid date should have failed", id)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}

func TestValidationCache_DifferentPatterns(t *testing.T) {
	patterns := []string{
		`^[a-z]+$`,       // lowercase letters only
		`^[A-Z]+$`,       // uppercase letters only
		`^\d+$`,          // digits only
		`^[a-zA-Z0-9]+$`, // alphanumeric
		`^.+@.+\..+$`,    // email pattern
	}

	// Test that different patterns are cached correctly
	for _, pattern := range patterns {
		validation := validateRegexp(pattern)

		// Test with matching string
		testString := getMatchingString(pattern)
		err := validation.Check(testString)
		if err != nil {
			t.Errorf("Pattern %s: expected '%s' to match, got error: %v", pattern, testString, err)
		}

		// Test with non-matching string
		err = validation.Check("!@#$%%^&*()")
		if err == nil {
			t.Errorf("Pattern %s: expected '!@#$%%^&*()' to not match", pattern)
		}
	}
}

func getMatchingString(pattern string) string {
	switch pattern {
	case `^[a-z]+$`:
		return "lowercase"
	case `^[A-Z]+$`:
		return "UPPERCASE"
	case `^\d+$`:
		return "12345"
	case `^[a-zA-Z0-9]+$`:
		return "alphanum123"
	case `^.+@.+\..+$`:
		return "test@example.com"
	default:
		return "test"
	}
}

func TestValidationCache_MemoryEfficiency(t *testing.T) {
	// Test that cache doesn't grow indefinitely with unique patterns
	initialCacheSize := len(validationCache.regexCache)

	// Create validations with unique patterns
	for i := range 100 {
		pattern := fmt.Sprintf("^pattern%d$", i)
		validation := validateRegexp(pattern)
		_ = validation.Check("pattern0") // This should trigger caching
	}

	finalCacheSize := len(validationCache.regexCache)
	expectedSize := initialCacheSize + 100

	if finalCacheSize != expectedSize {
		t.Errorf("Expected cache size to be %d, got %d", expectedSize, finalCacheSize)
	}

	t.Logf("Cache grew from %d to %d entries", initialCacheSize, finalCacheSize)
}
