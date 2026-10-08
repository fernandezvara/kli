// cli/files_test.go
package kli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPriorityResolution(t *testing.T) {
	t.Setenv("PRIORITY_TEST", "env_value")

	cfg := New()

	// Test Flag > Env > Default priority
	cfg.Define("TEST_FLAG_PRIORITY").
		String().
		Env("PRIORITY_TEST").
		Flag("test-flag").
		Default("default_value").
		Priority(PriorityFlagEnvDefault)

	if err := cfg.Execute([]string{"test", "--test-flag", "flag_value"}); err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Flag should win with Flag > Env > Default priority
	value := cfg.values["TEST_FLAG_PRIORITY"]
	if value != "flag_value" {
		t.Errorf("Expected flag_value, got %s", value)
	}
}

func TestEnvOverFlagPriority(t *testing.T) {
	t.Setenv("PRIORITY_TEST2", "env_value")

	cfg := New()

	// Test Env > Flag > Default priority
	cfg.Define("TEST_ENV_PRIORITY").
		String().
		Env("PRIORITY_TEST2").
		Flag("test-flag2").
		Default("default_value").
		Priority(PriorityEnvFlagDefault)

	if err := cfg.Execute([]string{"test", "--test-flag2", "flag_value"}); err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Env should win with Env > Flag > Default priority
	value := cfg.values["TEST_ENV_PRIORITY"]
	if value != "env_value" {
		t.Errorf("Expected env_value, got %s", value)
	}
}

// TestFileSourceResolvesEndToEnd verifies that a definition with File() resolves
// its value from a loaded config file when no flag/env/default applies.
func TestFileSourceResolvesEndToEnd(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"endpoint_url": "https://file-source.example.com"}`), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := New()
	if err := cfg.LoadFile(cfgFile); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// File-keyed definition: no flag, no env, no default - only file source
	cfg.Define("ENDPOINT").
		String().
		File("endpoint_url")

	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")
	val, err := Get[string](ctx, "ENDPOINT")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "https://file-source.example.com" {
		t.Errorf("Expected file-sourced value, got %q", val)
	}
}

// TestFileSourcePriorityOrder verifies Flag > Env > File > Default ordering.
func TestFileSourcePriorityOrder(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"val": "from_file"}`), 0644); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// Case 1: file wins over default
	cfg := New()
	if err := cfg.LoadFile(cfgFile); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	cfg.Define("VAL").String().File("val").Default("from_default")
	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got := cfg.values["VAL"]; got != "from_file" {
		t.Errorf("Expected 'from_file' (file > default), got %v", got)
	}

	// Case 2: env wins over file
	cfg2 := New()
	if err := cfg2.LoadFile(cfgFile); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	t.Setenv("TEST_PRIO_VAL", "from_env")
	cfg2.Define("VAL").String().Env("TEST_PRIO_VAL").File("val").Default("from_default")
	if err := cfg2.Execute([]string{"test"}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got := cfg2.values["VAL"]; got != "from_env" {
		t.Errorf("Expected 'from_env' (env > file), got %v", got)
	}

	// Case 3: flag wins over env and file
	cfg3 := New()
	if err := cfg3.LoadFile(cfgFile); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	cfg3.Define("VAL").String().Env("TEST_PRIO_VAL").Flag("val").File("val").Default("from_default")
	if err := cfg3.Execute([]string{"test", "--val", "from_flag"}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got := cfg3.values["VAL"]; got != "from_flag" {
		t.Errorf("Expected 'from_flag' (flag > env > file), got %v", got)
	}
}
