// cli/errors.go
package kli

import (
	"fmt"
	"strings"
)

func shouldDisplayDefault(def *Definition) bool {
	if def.defaultValue == nil {
		return false
	}
	if value, ok := def.defaultValue.(string); ok && value == "" {
		return false // an empty default just means "optional"
	}
	if def.valueType == TypeBool {
		if value, ok := def.defaultValue.(bool); ok && !value {
			return false
		}
	}
	return true
}

// ConfigError represents a single configuration error
type ConfigError struct {
	Key              string
	Source           string // "env", "flag", "default", or "none"
	Value            string // Masked if secret
	Display          string
	ErrorDescription string
}

func (e *ConfigError) Error() string {
	return e.ErrorDescription
}

// buildErrorDisplay creates the display string for a definition in error output.
// Delegates to buildDefinitionDisplay so errors and help render identically.
func buildErrorDisplay(def *Definition) string {
	return buildDefinitionDisplay(def)
}

// buildDefinitionDisplay creates unified display for both flags and environment variables
func buildDefinitionDisplay(def *Definition) string {
	valueType := def.valueType.String()
	var indicators []string

	// Collect all indicators
	if shouldDisplayDefault(def) {
		indicators = append(indicators, fmt.Sprintf("default: %v", def.defaultValue))
	}
	if def.required {
		indicators = append(indicators, "required")
	}
	if def.secret {
		indicators = append(indicators, "secret")
	}

	// Add validations
	validations := formatValidation(def.validations)
	indicators = append(indicators, validations...)

	// Determine the base format based on what type of display this is
	var base string
	if def.flag != "" {
		// This is a flag display
		base = fmt.Sprintf("--%s %s", def.flag, valueType)

		// Add env indicator if it also has an environment variable
		if def.envVar != "" {
			indicators = append(indicators, fmt.Sprintf("env: %s", def.envVar))
		}
	} else if def.envVar != "" {
		// This is an environment-only variable display
		base = fmt.Sprintf("%s %s", def.envVar, valueType)
		// Note: Don't add "env: VARNAME" since this IS the env var display
	} else {
		// No flag or env var - fall back to the definition key
		base = fmt.Sprintf("%s %s", def.key, valueType)
	}

	// Return with indicators if any
	if len(indicators) == 0 {
		return base
	}

	return fmt.Sprintf("%s (%s)", base, strings.Join(indicators, ", "))
}

// newConfigError creates a unified ConfigError for all error types
func newConfigError(key string, def *Definition, source string, rawValue string, original error) ConfigError {
	return ConfigError{
		Key:              key,
		Source:           source,
		Value:            rawValue,
		Display:          buildErrorDisplay(def),
		ErrorDescription: original.Error(),
	}
}

// maskSecret masks a secret value for display
func maskSecret(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
