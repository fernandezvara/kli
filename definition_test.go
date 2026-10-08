package kli

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDefinitionBuilderTypes(t *testing.T) {
	cfg := New()

	// Test all type setters
	cfg.Define("STRING_VAL").String()
	cfg.Define("INT64_VAL").Int64()
	cfg.Define("FLOAT64_VAL").Float64()
	cfg.Define("BOOL_VAL").Bool()
	cfg.Define("DURATION_VAL").Duration()
	cfg.Define("STRING_SLICE_VAL").StringSlice()
	cfg.Define("INT64_SLICE_VAL").Int64Slice()

	// Verify types are set correctly
	if cfg.definitions["STRING_VAL"].valueType != TypeString {
		t.Error("String() should set TypeString")
	}
	if cfg.definitions["INT64_VAL"].valueType != TypeInt64 {
		t.Error("Int64() should set TypeInt64")
	}
	if cfg.definitions["FLOAT64_VAL"].valueType != TypeFloat64 {
		t.Error("Float64() should set TypeFloat64")
	}
	if cfg.definitions["BOOL_VAL"].valueType != TypeBool {
		t.Error("Bool() should set TypeBool")
	}
	if cfg.definitions["DURATION_VAL"].valueType != TypeDuration {
		t.Error("Duration() should set TypeDuration")
	}
	if cfg.definitions["STRING_SLICE_VAL"].valueType != TypeStringSlice {
		t.Error("StringSlice() should set TypeStringSlice")
	}
	if cfg.definitions["INT64_SLICE_VAL"].valueType != TypeInt64Slice {
		t.Error("Int64Slice() should set TypeInt64Slice")
	}
}

func TestDefinitionBuilderSources(t *testing.T) {
	cfg := New()

	cfg.Define("PORT").Int64().Env("PORT_ENV").Flag("port-flag")

	def := cfg.definitions["PORT"]
	if def.envVar != "PORT_ENV" {
		t.Errorf("Env() should set envVar, got %s", def.envVar)
	}
	if def.flag != "port-flag" {
		t.Errorf("Flag() should set flag, got %s", def.flag)
	}
}

func TestDefinitionBuilderBehaviors(t *testing.T) {
	cfg := New()

	cfg.Define("REQUIRED_VAL").String().Required()
	cfg.Define("SECRET_VAL").String().Secret()
	cfg.Define("DEFAULT_VAL").String().Default("default-value")
	cfg.Define("DELIM_VAL").StringSlice().Delimiter("|")
	cfg.Define("DESC_VAL").String().Description("Test description")

	if !cfg.definitions["REQUIRED_VAL"].required {
		t.Error("Required() should set required=true")
	}
	if !cfg.definitions["SECRET_VAL"].secret {
		t.Error("Secret() should set secret=true")
	}
	if cfg.definitions["DEFAULT_VAL"].defaultValue != "default-value" {
		t.Error("Default() should set defaultValue")
	}
	if cfg.definitions["DELIM_VAL"].delimiter != "|" {
		t.Error("Delimiter() should set delimiter")
	}
	if cfg.definitions["DESC_VAL"].description != "Test description" {
		t.Error("Description() should set description")
	}
}

func TestDefinitionBuilderNumericValidation(t *testing.T) {
	cfg := New()

	cfg.Define("MIN_VAL").Int64().Min(10)
	cfg.Define("MAX_VAL").Int64().Max(100)
	cfg.Define("RANGE_VAL").Int64().Range(1, 65535)

	// Min validation
	if len(cfg.definitions["MIN_VAL"].validations) != 1 {
		t.Error("Min() should add 1 validation")
	}

	// Max validation
	if len(cfg.definitions["MAX_VAL"].validations) != 1 {
		t.Error("Max() should add 1 validation")
	}

	// Range validation (adds 2: min and max)
	if len(cfg.definitions["RANGE_VAL"].validations) != 2 {
		t.Error("Range() should add 2 validations")
	}
}

func TestDefinitionBuilderStringValidation(t *testing.T) {
	cfg := New()

	cfg.Define("MIN_LEN").String().MinLength(5)
	cfg.Define("MAX_LEN").String().MaxLength(100)
	cfg.Define("LEN_RANGE").String().MinLength(5).MaxLength(100)
	cfg.Define("REGEXP").String().Regexp(`^[a-z]+$`)
	cfg.Define("ONE_OF").String().OneOf("a", "b", "c")

	if len(cfg.definitions["MIN_LEN"].validations) != 1 {
		t.Error("MinLength() should add 1 validation")
	}
	if len(cfg.definitions["MAX_LEN"].validations) != 1 {
		t.Error("MaxLength() should add 1 validation")
	}
	if len(cfg.definitions["LEN_RANGE"].validations) != 2 {
		t.Error("MinLength()+MaxLength() should add 2 validations")
	}
	if len(cfg.definitions["REGEXP"].validations) != 1 {
		t.Error("Regexp() should add 1 validation")
	}
	if len(cfg.definitions["ONE_OF"].validations) != 1 {
		t.Error("OneOf() should add 1 validation")
	}
}

func TestDefinitionBuilderDurationValidation(t *testing.T) {
	cfg := New()

	cfg.Define("MIN_DUR").Duration().MinDuration(5 * time.Second)
	cfg.Define("MAX_DUR").Duration().MaxDuration(1 * time.Minute)
	cfg.Define("DUR_RANGE").Duration().MinDuration(5 * time.Second).MaxDuration(1 * time.Minute)

	if len(cfg.definitions["MIN_DUR"].validations) != 1 {
		t.Error("MinDuration() should add 1 validation")
	}
	if len(cfg.definitions["MAX_DUR"].validations) != 1 {
		t.Error("MaxDuration() should add 1 validation")
	}
	if len(cfg.definitions["DUR_RANGE"].validations) != 2 {
		t.Error("MinDuration()+MaxDuration() should add 2 validations")
	}
}

func TestDefinitionBuilderArrayValidation(t *testing.T) {
	cfg := New()

	cfg.Define("MIN_ITEMS").StringSlice().MinItems(2)
	cfg.Define("MAX_ITEMS").StringSlice().MaxItems(5)
	cfg.Define("ITEMS_RANGE").StringSlice().MinItems(2).MaxItems(5)

	if len(cfg.definitions["MIN_ITEMS"].validations) != 1 {
		t.Error("MinItems() should add 1 validation")
	}
	if len(cfg.definitions["MAX_ITEMS"].validations) != 1 {
		t.Error("MaxItems() should add 1 validation")
	}
	if len(cfg.definitions["ITEMS_RANGE"].validations) != 2 {
		t.Error("MinItems()+MaxItems() should add 2 validations")
	}
}

func TestDefinitionBuilderCustomValidation(t *testing.T) {
	cfg := New()

	evenLength := func(value any) error {
		if s, ok := value.(string); ok && len(s)%2 != 0 {
			return fmt.Errorf("string length must be even")
		}
		return nil
	}

	cfg.Define("CUSTOM").String().Custom("even-length", evenLength).Default("ab")

	if len(cfg.definitions["CUSTOM"].validations) != 1 {
		t.Error("Custom() should add 1 validation")
	}

	// Test that custom validation works
	if err := cfg.Execute([]string{"test"}); err != nil {
		t.Errorf("Unexpected errors for valid value: %v", err)
	}

	cfg2 := New()
	cfg2.Define("CUSTOM").String().Custom("even-length", evenLength).Default("abc")
	if err := cfg2.Execute([]string{"test"}); err == nil {
		t.Errorf("Expected error for odd-length string, got none")
	}
}

func TestDefinitionBuilderChaining(t *testing.T) {
	cfg := New()

	// Test that all methods can be chained
	cfg.Define("FULL").
		String().
		Env("FULL_ENV").
		Flag("full-flag").
		Default("default").
		Required().
		MinLength(3).
		MaxLength(100).
		Description("Full test")

	def := cfg.definitions["FULL"]
	if def.valueType != TypeString {
		t.Error("Chaining: type not set")
	}
	if def.envVar != "FULL_ENV" {
		t.Error("Chaining: envVar not set")
	}
	if def.flag != "full-flag" {
		t.Error("Chaining: flag not set")
	}
	if def.defaultValue != "default" {
		t.Error("Chaining: defaultValue not set")
	}
	if !def.required {
		t.Error("Chaining: required not set")
	}
	if def.description != "Full test" {
		t.Error("Chaining: description not set")
	}
	if len(def.validations) != 2 { // minLength + maxLength
		t.Errorf("Chaining: expected 2 validations, got %d", len(def.validations))
	}
}

func TestFormatValidation(t *testing.T) {
	validations := []Validation{
		{Name: "min(1)"},
		{Name: "max(65535)"},
		{Name: "oneOf(debug,info,warn,error)"},
		{Name: "minLength(3)"},
		{Name: "regexp(^[a-z]+$)"},
		{Name: "custom-check"},
	}

	formatted := formatValidation(validations)
	joined := strings.Join(formatted, " | ")

	if !strings.Contains(joined, "valid: 1-65535") {
		t.Fatalf("expected range formatting, got: %v", formatted)
	}
	if !strings.Contains(joined, "oneOf: ['debug', 'info', 'warn', 'error']") {
		t.Fatalf("expected oneOf formatting, got: %v", formatted)
	}
	if !strings.Contains(joined, "minLength: 3") {
		t.Fatalf("expected minLength formatting, got: %v", formatted)
	}
	if !strings.Contains(joined, "pattern: ^[a-z]+$") {
		t.Fatalf("expected regexp formatting, got: %v", formatted)
	}
	if !strings.Contains(joined, "custom-check") {
		t.Fatalf("expected passthrough validation name, got: %v", formatted)
	}
}

func TestExtractValue(t *testing.T) {
	if got := extractValue("min(8080)", "min("); got != "8080" {
		t.Fatalf("expected 8080, got %q", got)
	}
	if got := extractValue("regexp(^[a-z]+$)", "regexp("); got != "^[a-z]+$" {
		t.Fatalf("expected regexp body, got %q", got)
	}
	if got := extractValue("other(1)", "min("); got != "" {
		t.Fatalf("expected empty string for missing prefix, got %q", got)
	}
	if got := extractValue("min(123", "min("); got != "123" {
		t.Fatalf("expected unterminated extraction fallback, got %q", got)
	}
}

func TestExtractOneOfValues(t *testing.T) {
	if got := extractOneOfValues("oneOf(debug,info,warn,error)"); got != "['debug', 'info', 'warn', 'error']" {
		t.Fatalf("unexpected oneOf comma formatting: %q", got)
	}
	if got := extractOneOfValues("oneOf([debug info warn error])"); got != "['debug', 'info', 'warn', 'error']" {
		t.Fatalf("unexpected oneOf array formatting: %q", got)
	}
	if got := extractOneOfValues("missing"); got != "" {
		t.Fatalf("expected empty string for invalid format, got %q", got)
	}
}

func TestDefinitionBuilderBuild(t *testing.T) {
	cfg := New()

	builder := newDefinitionBuilder(cfg, "TEST")
	builder.String().Default("test")

	def := builder.def
	if def.key != "TEST" {
		t.Error("build() should return definition with correct key")
	}
	if def.defaultValue != "test" {
		t.Error("build() should return definition with correct default")
	}
}

func TestDefinitionPriorityMethods(t *testing.T) {
	cfg := New()

	// Test Priority() method
	def := cfg.Define("TEST2").String().Priority(PriorityEnvFlagDefault)
	if len(def.def.priority) != 3 {
		t.Error("Priority should set 3 elements")
	}
	if def.def.priority[0] != SourceEnv || def.def.priority[1] != SourceFlag || def.def.priority[2] != SourceDefault {
		t.Error("Priority should be [Env, Flag, Default]")
	}

	def = cfg.Define("TEST3").String().Priority(PriorityFlagEnvDefault)
	if len(def.def.priority) != 3 {
		t.Error("Priority(PriorityFlagEnvDefault) should set 3 elements")
	}
	if def.def.priority[0] != SourceFlag || def.def.priority[1] != SourceEnv || def.def.priority[2] != SourceDefault {
		t.Error("Priority should be [Flag, Env, Default]")
	}
}

func TestEffectivePriority(t *testing.T) {
	cfg := New()

	// Definition with explicit priority
	def1 := &Definition{
		priority: PriorityEnvFlagDefault,
	}
	effective1 := def1.getEffectivePriority(cfg.defaultPriority)
	if len(effective1) != 3 || effective1[0] != SourceEnv {
		t.Error("Should use definition's priority when set")
	}

	// Definition without explicit priority
	def2 := &Definition{
		priority: nil,
	}
	effective2 := def2.getEffectivePriority(cfg.defaultPriority)
	if len(effective2) != 4 || effective2[0] != SourceFlag {
		t.Error("Should use config default when definition priority is nil")
	}
}

func TestInferAvailableSources(t *testing.T) {
	// Definition with all sources
	def1 := &Definition{
		envVar:       "TEST_ENV",
		flag:         "test-flag",
		defaultValue: "default",
	}
	sources1 := def1.inferAvailableSources()
	if len(sources1) != 4 {
		t.Error("Should infer 4 sources when all are available")
	}

	// Definition with only env
	def2 := &Definition{
		envVar: "TEST_ENV",
	}
	sources2 := def2.inferAvailableSources()
	if len(sources2) != 2 {
		t.Error("Should infer 2 sources (env + file) when only env is set")
	}

	// Definition with only default
	def3 := &Definition{
		defaultValue: "default",
	}
	sources3 := def3.inferAvailableSources()
	if len(sources3) != 2 {
		t.Error("Should infer 2 sources (default + file) when only default is set")
	}
}

func TestIntTypeSupport(t *testing.T) {
	cfg := New()

	// Test Int64() method
	cfg.Define("INT_PORT").
		Int64().
		Flag("port").
		Default(8080)

	// Test Int64Slice() method
	cfg.Define("INT_PORTS").
		Int64Slice().
		Flag("ports").
		Default([]int64{8080, 8081})

	// Verify the types are set correctly
	if def, exists := cfg.definitions["INT_PORT"]; exists {
		if def.valueType != TypeInt64 {
			t.Errorf("INT_PORT type = %v, expected TypeInt64", def.valueType)
		}
	} else {
		t.Error("INT_PORT definition not found")
	}

	if def, exists := cfg.definitions["INT_PORTS"]; exists {
		if def.valueType != TypeInt64Slice {
			t.Errorf("INT_PORTS type = %v, expected TypeInt64Slice", def.valueType)
		}
	} else {
		t.Error("INT_PORTS definition not found")
	}
}

// TestPriorityValidationErrorSurfaced verifies that an invalid Priority()
// (referencing a source that isn't available on the definition) surfaces
// as an Execute() error naming the definition key.
func TestPriorityValidationErrorSurfaced(t *testing.T) {
	cfg := New()

	// Definition has no flag, so Priority{SourceFlag} is invalid
	cfg.Define("BROKEN").String().Env("BROKEN_ENV").Default("x").Priority(SourcePriority{SourceFlag, SourceDefault})

	// Key-level check: processDefinitions must surface a ConfigError for "BROKEN"
	errs := cfg.processDefinitions()
	found := false
	for _, e := range errs {
		if e.Key == "BROKEN" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected processDefinitions to surface error for definition key 'BROKEN'")
	}

	// Integration-level check: Execute must fail
	err := cfg.Execute([]string{"test"})
	if err == nil {
		t.Fatal("Expected Execute to return an error for invalid priority")
	}
}
