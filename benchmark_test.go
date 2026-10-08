// cli/benchmark_test.go
package kli

import (
	"regexp"
	"testing"
	"time"
)

// BenchmarkGetTypeDescription benchmarks the type description caching
func BenchmarkGetTypeDescription(b *testing.B) {
	for b.Loop() {
		typeDescription("test")
		typeDescription(int64(0))
		typeDescription(int(0))
		typeDescription(true)
		typeDescription(float64(0))
		typeDescription([]string{"a", "b"})
		typeDescription([]int64{1, 2})
		typeDescription([]int{1, 2})
	}
}

// BenchmarkTypeConverter benchmarks the new TypeConverter
func BenchmarkTypeConverter(b *testing.B) {
	converter := NewTypeConverter()

	b.Run("string", func(b *testing.B) {
		for b.Loop() {
			_, _ = converter.ConvertToString("hello", ",")
		}
	})

	b.Run("string_slice", func(b *testing.B) {
		slice := []string{"a", "b", "c"}
		for b.Loop() {
			_, _ = converter.ConvertToString(slice, ",")
		}
	})

	b.Run("int64_slice", func(b *testing.B) {
		slice := []int64{1, 2, 3}
		for b.Loop() {
			_, _ = converter.ConvertToString(slice, ",")
		}
	})

	b.Run("int_slice", func(b *testing.B) {
		slice := []int{1, 2, 3}
		for b.Loop() {
			_, _ = converter.ConvertToString(slice, ",")
		}
	})
}

// BenchmarkGetInt64 benchmarks Get[int64] performance
func BenchmarkGetInt64(b *testing.B) {
	cfg := New()
	cfg.Define("PORT").Int64().Default(8080)

	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	for b.Loop() {
		_, err := Get[int64](ctx, "PORT")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetString benchmarks Get[string] performance
func BenchmarkGetString(b *testing.B) {
	cfg := New()
	cfg.Define("HOST").String().Default("localhost")

	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	for b.Loop() {
		_, err := Get[string](ctx, "HOST")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetBool benchmarks Get[bool] performance
func BenchmarkGetBool(b *testing.B) {
	cfg := New()
	cfg.Define("DEBUG").Bool().Default(false)

	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	for b.Loop() {
		_, err := Get[bool](ctx, "DEBUG")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetStringSlice benchmarks Get[[]string] performance
func BenchmarkGetStringSlice(b *testing.B) {
	cfg := New()
	cfg.Define("TAGS").StringSlice().Default([]string{"tag1", "tag2"})

	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	for b.Loop() {
		_, err := Get[[]string](ctx, "TAGS")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetInt64Slice benchmarks Get[[]int64] performance
func BenchmarkGetInt64Slice(b *testing.B) {
	cfg := New()
	cfg.Define("NUMBERS").Int64Slice().Default([]int64{1, 2, 3})

	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	for b.Loop() {
		_, err := Get[[]int64](ctx, "NUMBERS")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidationRegex_Cached benchmarks regexp validation with the cache
func BenchmarkValidationRegex_Cached(b *testing.B) {
	pattern := `^[a-z]+@[a-z]+\.[a-z]+$`
	validation := validateRegexp(pattern)

	for b.Loop() {
		_ = validation.Check("test@example.com")
	}
}

// BenchmarkValidationRegex_Uncached benchmarks compiling the regexp on each call
func BenchmarkValidationRegex_Uncached(b *testing.B) {
	pattern := `^[a-z]+@[a-z]+\.[a-z]+$`

	for b.Loop() {
		// This simulates the old behavior - compiling regex each time
		re := regexp.MustCompile(pattern)
		re.MatchString("test@example.com")
	}
}

// BenchmarkConfigProcessing_Large benchmarks configuration processing with many definitions
func BenchmarkConfigProcessing_Large(b *testing.B) {
	for b.Loop() {
		cfg := New()

		// Create many definitions to simulate real-world usage
		for range 100 {
			cfg.Define("PORT").
				Int64().
				Env("PORT").
				Flag("port").
				Default(8080)

			cfg.Define("HOST").
				String().
				Env("HOST").
				Flag("host").
				Default("localhost")

			cfg.Define("DEBUG").
				Bool().
				Env("DEBUG").
				Flag("debug").
				Default(false)

			cfg.Define("RATE").
				Float64().
				Env("RATE").
				Flag("rate").
				Default(100.0).
				Range(1.0, 1000.0)

			cfg.Define("TIMEOUT").
				Duration().
				Env("TIMEOUT").
				Flag("timeout").
				Default(30 * time.Second)
		}

		// Process configuration
		if err := cfg.Execute([]string{"test"}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfigProcessing_Small benchmarks configuration processing with few definitions
func BenchmarkConfigProcessing_Small(b *testing.B) {
	for b.Loop() {
		cfg := New()

		cfg.Define("PORT").
			Int64().
			Env("PORT").
			Flag("port").
			Default(8080)

		cfg.Define("HOST").
			String().
			Env("HOST").
			Flag("host").
			Default("localhost")

		// Process configuration
		if err := cfg.Execute([]string{"test"}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHelpGeneration_Global benchmarks global help generation performance
func BenchmarkHelpGeneration_Global(b *testing.B) {
	cfg := New()

	// Create multiple commands
	for range 10 {
		cfg.Command("start").
			Func(func(ctx *CommandContext) error { return nil }).
			ShortHelp("Start the service").
			LongHelp("Start the service with all components initialized.")

		cfg.Command("stop").
			Func(func(ctx *CommandContext) error { return nil }).
			ShortHelp("Stop the service").
			LongHelp("Stop the service gracefully.")

		cfg.Command("status").
			Func(func(ctx *CommandContext) error { return nil }).
			ShortHelp("Show service status").
			LongHelp("Display current service status and statistics.")
	}

	// Process configuration
	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		// Generate help
		help := cfg.GenerateHelp()
		if help == "" {
			b.Fatal("Empty help generated")
		}
	}
}

// BenchmarkHelpGeneration_Command benchmarks command-specific help generation
func BenchmarkHelpGeneration_Command(b *testing.B) {
	cfg := New()

	// Create a command with many definitions
	cfg.Command("deploy").
		Func(func(ctx *CommandContext) error { return nil }).
		ShortHelp("Deploy the application").
		LongHelp("Deploy the application to the specified environment.").
		Config(func(cc *CommandConfig) {
			for range 20 {
				cc.Define("PORT").
					Int64().
					Flag("port").
					Default(8080)

				cc.Define("ENV").
					String().
					Flag("env").
					Default("development")

				cc.Define("DRY_RUN").
					Bool().
					Flag("dry-run").
					Default(false)
			}
		})

	// Process configuration
	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		// Generate command help
		err := cfg.ShowCommandHelp("deploy")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidation_MultipleTypes benchmarks validation with different types
func BenchmarkValidation_MultipleTypes(b *testing.B) {
	cfg := New()

	// Define various validation types
	cfg.Define("PORT").
		Int64().
		Range(1, 65535).
		Default(8080)

	cfg.Define("EMAIL").
		String().
		Regexp(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).
		Default("test@example.com")

	cfg.Define("RATE").
		Float64().
		Range(0.1, 100.0).
		Default(1.0)

	cfg.Define("TOKEN").
		String().
		MinLength(32).
		MaxLength(64).
		Default("12345678901234567890123456789012")

	cfg.Define("ENVIRONMENTS").
		StringSlice().
		MinItems(1).
		MaxItems(5).
		Default([]string{"dev", "staging", "prod"})

	cfg.Define("TIMEOUT").
		Duration().
		MinDuration(1 * time.Second).
		MaxDuration(5 * time.Minute).
		Default(30 * time.Second)

	// Process configuration
	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	for b.Loop() {
		// Get all values to trigger validation
		_, _ = Get[int64](ctx, "PORT")
		_, _ = Get[string](ctx, "EMAIL")
		_, _ = Get[float64](ctx, "RATE")
		_, _ = Get[string](ctx, "TOKEN")
		_, _ = Get[[]string](ctx, "ENVIRONMENTS")
		_, _ = Get[time.Duration](ctx, "TIMEOUT")
	}
}

// BenchmarkErrorFormatting benchmarks error message formatting
func BenchmarkErrorFormatting(b *testing.B) {
	cfg := New()

	cfg.Define("PORT").
		Int64().
		Range(1, 65535).
		Default(8080)

	cfg.Define("EMAIL").
		String().
		Regexp(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).
		Default("test@example.com")

	// Process configuration
	if err := cfg.Execute([]string{"test"}); err != nil {
		b.Fatal(err)
	}

	ctx := NewCommandContext([]string{}, cfg, "test", "")

	// Create execution context with errors
	execCtx := NewExecutionContext("test")

	// Add some config errors
	configErr := ConfigError{
		Key:              "PORT",
		Source:           "flag",
		Value:            "99999",
		ErrorDescription: "value 99999 is greater than maximum 65535",
	}

	execCtx.CollectConfigError(cfg, configErr)

	ctx.execution = execCtx

	for b.Loop() {
		// Format error with template
		_, err := ctx.execution.renderErrorsWithCommand(nil, cfg.getHelpService())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFileOperations benchmarks file loading and processing
func BenchmarkFileOperations(b *testing.B) {
	for b.Loop() {
		cfg := New()
		cfg.SetDefaultPriority(PriorityFileEnvFlagDefault)

		cfg.Define("PORT").
			Int64().
			File("port_in_file").
			Default(8080)

		cfg.Define("HOST").
			String().
			File("host_in_file").
			Default("localhost")

		cfg.Define("DEBUG").
			Bool().
			File("debug_in_file").
			Default(false)

		cfg.Define("RATE").
			Float64().
			File("rate_in_file").
			Default(100.0)

		cfg.Define("TIMEOUT").
			Duration().
			File("timeout_in_file").
			Default(30 * time.Second)

		// Simulate file loading (in real usage this would be cfg.LoadFile())
		if cfg.fileConfig == nil {
			cfg.fileConfig = &FileConfig{
				data: map[string]any{
					"port_in_file":  3000.0,
					"host_in_file":  "localhost",
					"debug_in_file": true,
					"rate_in_file":  100.5,
					"environments": map[string]any{
						"development": map[string]any{
							"timeout_in_file": "30s",
						},
						"production": map[string]any{
							"timeout_in_file": "10s",
						},
					},
				},
			}
		}

		// Process configuration
		if err := cfg.Execute([]string{"test"}); err != nil {
			b.Fatal(err)
		}
	}
}
