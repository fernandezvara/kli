// cli Type Conversion TDD Test Suite
// Tests all supported type conversions across all sources (DEFAULT, ENV, FLAG, FILE)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fernandezvara/kli"
)

// TestResult holds one test outcome
type TestResult struct {
	Source       string
	Key          string
	ExpectedType string
	ReturnedType string
	Value        string
	Success      bool
	Error        string
}

var allResults []TestResult

func addResult(source, key, expectedType, returnedType, value string, success bool, errStr string) {
	allResults = append(allResults, TestResult{
		Source:       source,
		Key:          key,
		ExpectedType: expectedType,
		ReturnedType: returnedType,
		Value:        value,
		Success:      success,
		Error:        errStr,
	})
}

// testGet is a generic helper that tests Get[T] and records the result
func testGet[T any](source, key, expectedType string, ctx *kli.CommandContext) {
	val, err := kli.Get[T](ctx, key)
	if err != nil {
		addResult(source, key, expectedType, "error", "", false, err.Error())
		return
	}
	valStr := fmt.Sprintf("%v", val)
	retType := fmt.Sprintf("%T", val)
	addResult(source, key, expectedType, retType, valStr, true, "")
}

// runAllTypeTests runs Get[T] for every supported type using the given context
func runAllTypeTests(source string, ctx *kli.CommandContext) {
	// Basic types
	testGet[string](source, "TEST_STRING", "string", ctx)
	testGet[bool](source, "TEST_BOOL", "bool", ctx)
	testGet[int64](source, "TEST_INT64", "int64", ctx)
	testGet[float64](source, "TEST_FLOAT64", "float64", ctx)
	testGet[time.Duration](source, "TEST_DURATION", "time.Duration", ctx)

	// Slice types
	testGet[[]string](source, "TEST_STRING_SLICE", "[]string", ctx)
	testGet[[]int64](source, "TEST_INT64_SLICE", "[]int64", ctx)
}

// defineAllTypes adds all type definitions inside a command config callback
func defineAllTypes(cc *kli.CommandConfig, withEnv, withFlag, withFile bool) {
	d := func(key string) *kli.DefinitionBuilder {
		b := cc.Define(key)
		if withEnv {
			b.Env(key)
		}
		if withFlag {
			b.Flag(strings.ToLower(key))
		}
		if withFile {
			b.File(strings.ToLower(key))
		}
		return b
	}

	// Basic types
	d("TEST_STRING").String().Default("default_string")
	d("TEST_BOOL").Bool().Default(true)
	d("TEST_INT64").Int64().Default(1)
	d("TEST_FLOAT64").Float64().Default(1.1)
	d("TEST_DURATION").Duration().Default(time.Minute)

	// Slice types
	d("TEST_STRING_SLICE").StringSlice().Default([]string{"d1", "d2"})
	d("TEST_INT64_SLICE").Int64Slice().Default([]int64{1, 2})
}

// testSourceWithCommand creates a config, adds a command that runs the tests, and executes it
func testSourceWithCommand(source string, withEnv, withFlag, withFile bool, extraArgs []string) {
	cfg := kli.New()

	if withFile {
		err := cfg.LoadFile(testConfigPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load test config file: %v\n", err)
		}
	}

	cfg.Command("test").
		Func(func(ctx *kli.CommandContext) error {
			runAllTypeTests(source, ctx)
			return nil
		}).
		ShortHelp("Run conversion tests").
		Config(func(cc *kli.CommandConfig) {
			defineAllTypes(cc, withEnv, withFlag, withFile)
		})

	args := append([]string{"conversions", "test"}, extraArgs...)
	err := cfg.Execute(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing %s source test: %v\n", source, err)
	}
}

// --- Environment variable helpers ---

func setupEnvVars() {
	os.Setenv("TEST_STRING", "env_string_value")
	os.Setenv("TEST_BOOL", "false")
	os.Setenv("TEST_INT64", "42")
	os.Setenv("TEST_FLOAT64", "3.14159")
	os.Setenv("TEST_DURATION", "5m")
	os.Setenv("TEST_STRING_SLICE", "e1,e2,e3")
	os.Setenv("TEST_INT64_SLICE", "10,20,30")
}

func clearEnvVars() {
	keys := []string{
		"TEST_STRING", "TEST_BOOL", "TEST_INT64", "TEST_FLOAT64", "TEST_DURATION",
		"TEST_STRING_SLICE", "TEST_INT64_SLICE",
	}
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

// --- File config helpers ---

func testConfigPath() string {
	return "/tmp/cli_test_config.json"
}

func setupFileConfig() {
	data := map[string]any{
		"test_string":       "file_string_value",
		"test_bool":         "false",
		"test_int64":        "84",
		"test_float64":      "6.28318",
		"test_duration":     "10m",
		"test_string_slice": "f1,f2,f3",
		"test_int64_slice":  "100,200,300",
	}
	b, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(testConfigPath(), b, 0644)
}

func cleanupFileConfig() {
	os.Remove(testConfigPath())
}

// --- Flag args builder ---

func flagArgs() []string {
	return []string{
		"--test_string", "flag_string_value",
		"--test_bool", "false",
		"--test_int64", "100",
		"--test_float64", "9.8696",
		"--test_duration", "15m",
		"--test_string_slice", "g1,g2,g3",
		"--test_int64_slice", "1000,2000,3000",
	}
}

// --- Output ---

func printResults() {
	fmt.Printf("\n%-7s | %-25s | %-15s | %-15s | %-20s | %s\n",
		"Source", "Key", "Expected Type", "Returned Type", "Value", "OK")
	fmt.Println(strings.Repeat("-", 110))

	for _, r := range allResults {
		ok := "OK"
		errInfo := ""
		if !r.Success {
			ok = "FAIL"
			errInfo = r.Error
			if len(errInfo) > 40 {
				errInfo = errInfo[:37] + "..."
			}
		}
		val := r.Value
		if len(val) > 18 {
			val = val[:15] + "..."
		}
		if errInfo != "" {
			fmt.Printf("%-7s | %-25s | %-15s | %-15s | %-20s | %s %s\n",
				r.Source, r.Key, r.ExpectedType, r.ReturnedType, val, ok, errInfo)
		} else {
			fmt.Printf("%-7s | %-25s | %-15s | %-15s | %-20s | %s\n",
				r.Source, r.Key, r.ExpectedType, r.ReturnedType, val, ok)
		}
	}
}

func printSummary() {
	total := len(allResults)
	passed := 0
	failed := 0
	for _, r := range allResults {
		if r.Success {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total:  %d\n", total)
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)

	if failed > 0 {
		fmt.Printf("\nFailed tests:\n")
		for _, r := range allResults {
			if !r.Success {
				fmt.Printf("  %-7s %-25s %s\n", r.Source, r.Key, r.Error)
			}
		}
	}
}

// --- Main ---

func main() {
	fmt.Println("=== cli Type Conversion TDD Test Suite ===")
	fmt.Println("Testing all type conversions across all sources (DEFAULT, ENV, FLAG, FILE)")

	// 1. DEFAULT source (no env, no flags, no file)
	clearEnvVars()
	testSourceWithCommand("DEFAULT", false, false, false, nil)

	// 2. ENV source
	setupEnvVars()
	testSourceWithCommand("ENV", true, false, false, nil)
	clearEnvVars()

	// 3. FLAG source
	testSourceWithCommand("FLAG", false, true, false, flagArgs())

	// 4. FILE source
	setupFileConfig()
	defer cleanupFileConfig()
	testSourceWithCommand("FILE", false, false, true, nil)

	// Output
	printResults()
	printSummary()

	// Exit code
	for _, r := range allResults {
		if !r.Success {
			os.Exit(1)
		}
	}
}
