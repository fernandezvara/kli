// cli/get.go
package kli

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// typeCache provides performance optimization for type descriptions
var typeCache = sync.Map{}

// GetError represents an error collected from Get functions
type GetError struct {
	Key              string
	ExpectedType     string
	ActualType       string
	Message          string
	IsSecret         bool
	Flag             string // Flag name (e.g., "port")
	EnvVar           string // Environment variable name (e.g., "PORT")
	Display          string
	ErrorDescription string
	config           *Config // Reference to config for definition lookup
}

// typeDescription returns cached type description for performance
func typeDescription(v any) string {
	typ := reflect.TypeOf(v)
	if cached, ok := typeCache.Load(typ); ok {
		return cached.(string)
	}

	desc := getTypeDescription(v)
	typeCache.Store(typ, desc)
	return desc
}

// getTypeDescription converts type to human-readable string
func getTypeDescription(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case int64:
		return "int64"
	case bool:
		return "bool"
	case float64:
		return "float64"
	case time.Duration:
		return "duration"
	case []string:
		return "[]string"
	case []int64:
		return "[]int64"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// newTypeError creates optimized type mismatch error
func newTypeError[T any](key string, value any) error {
	expectedType := typeDescription(*new(T))
	actualType := typeDescription(value)
	return fmt.Errorf("configuration '%s' type mismatch: expected %s, got %s",
		key, expectedType, actualType)
}

// convertValue handles type conversion for Get[T] compatibility
func convertValue(value any, targetType reflect.Type) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("cannot convert nil value")
	}

	sourceValue := reflect.ValueOf(value)
	sourceType := sourceValue.Type()

	// Handle string to int64 conversion
	if targetType == reflect.TypeFor[int64]() && sourceType == reflect.TypeFor[string]() {
		parsed, err := strconv.ParseInt(sourceValue.String(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid int64: %s", sourceValue.String())
		}
		return parsed, nil
	}

	// Handle int64 to float64 widening
	if targetType == reflect.TypeFor[float64]() && sourceType == reflect.TypeFor[int64]() {
		return float64(sourceValue.Int()), nil
	}

	return value, nil
}

// logWarningForDesigner logs warnings for CLI designers about configuration issues
func logWarningForDesigner(message string) {
	// Use log package with a distinct prefix for designer warnings
	log.Printf("[CONFIG WARNING] %s", message)
}

// getErrorDisplayName returns the display name matching help message format
func getErrorDisplayName(err GetError, c *Config) string {
	if c != nil {
		if def, hasDef := c.definitions[err.Key]; hasDef {
			return buildDefinitionDisplay(def)
		}
	}

	if err.Flag != "" {
		return fmt.Sprintf("--%s string", err.Flag)
	}
	if err.EnvVar != "" {
		return fmt.Sprintf("(no flag) string (env: %s)", err.EnvVar)
	}
	return fmt.Sprintf("%s string", err.Key)
}

// Get retrieves a configuration value with proper error handling
// This function performs early secret detection to prevent type assertion exposure
func Get[T any](ctx *CommandContext, key string) (T, error) {
	var zero T

	// Determine which config to use (command config takes precedence)
	var c *Config
	if ctx.CommandConfig != nil {
		// Check if the key is defined in command config
		if _, hasDef := ctx.CommandConfig.definitions[key]; hasDef {
			c = ctx.CommandConfig
		} else {
			// Key not in command config, check global config
			c = ctx.GlobalConfig
		}
	} else {
		c = ctx.GlobalConfig
	}

	// Early secret detection - check definition before accessing values
	def, hasDef := c.definitions[key]
	if hasDef && def.secret {
		// Secure error handling - no secret exposure
		ctx.execution.CollectError(c, key, "secret", "", "use GetSecret() instead", true)
		result := validationError(fmt.Sprintf("configuration '%s' is secret, use GetSecret() instead", key))
		return zero, result.Error
	}

	value, exists := c.values[key]
	if !exists {
		// Check if this is required data - if so, return validation error
		if hasDef && def.required {
			// Log warning for designer and return validation error
			logWarningForDesigner(fmt.Sprintf("Required key '%s' is not provided", key))
			return zero, fmt.Errorf("required configuration '%s' not provided", key)
		}
		// For non-required keys, collect error and return result
		ctx.execution.CollectError(c, key, "not found", "", "key not defined", false)
		return zero, fmt.Errorf("configuration '%s' not found", key)
	}

	// Additional safety check - ensure value is not a secret placeholder
	if strVal, ok := value.(string); ok && strVal == "[SECRET]" {
		// This should not happen with new implementation, but add safety check
		ctx.execution.CollectError(c, key, "secret", "", "use GetSecret() instead", true)
		result := validationError(fmt.Sprintf("configuration '%s' is secret, use GetSecret() instead", key))
		return zero, result.Error
	}

	// Try direct type assertion first
	result, ok := value.(T)
	if ok {
		return result, nil
	}

	// Try type conversion for compatible types
	targetType := reflect.TypeOf(zero)
	converted, err := convertValue(value, targetType)
	if err == nil {
		result, ok = converted.(T)
		if ok {
			return result, nil
		}
	}

	// Type conversion failed
	ctx.execution.CollectError(c, key, typeDescription(*new(T)), typeDescription(value), "type mismatch", false)
	return zero, newTypeError[T](key, value)
}

// MustGet retrieves a configuration value and panics on error
// Use when you expect the configuration to be valid and want to fail fast
func MustGet[T any](ctx *CommandContext, key string) T {
	result, err := Get[T](ctx, key)
	if err != nil {
		panic(err)
	}
	return result
}

// Has checks if a key exists and has a non-nil value
// Note: This function will return false for secret keys to prevent exposure
func (c *Config) Has(key string) bool {
	// Check if this is a secret key - if so, always return false to prevent exposure
	if def, hasDef := c.definitions[key]; hasDef && def.secret {
		return false
	}

	value, exists := c.values[key]
	return exists && value != nil
}

// GetSecret retrieves a secret value securely
func (c *Config) GetSecret(key string) *Secret {
	return c.secrets.Get(key)
}

// HasSecret checks if a secret exists and is set
func (c *Config) HasSecret(key string) bool {
	return c.secrets.Has(key)
}

// Keys returns all defined configuration keys
func (c *Config) Keys() []string {
	keys := make([]string, 0, len(c.definitions))
	for k := range c.definitions {
		keys = append(keys, k)
	}
	return keys
}
