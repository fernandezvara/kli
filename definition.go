// cli/definition.go
package kli

import (
	"fmt"
	"strings"
	"time"
)

// Definition represents a configuration definition with fluent builder
type Definition struct {
	key          string
	valueType    ValueType
	envVar       string
	flag         string
	fileKey      string // Key name to look for in loaded files
	defaultValue any
	required     bool
	secret       bool
	delimiter    string
	validations  []Validation
	description  string
	priority     SourcePriority // Custom priority order (nil = use config default)
}

// DefinitionBuilder provides a fluent API for building definitions
type DefinitionBuilder struct {
	def    *Definition
	config *Config
}

// formatValidation formats validation rules for help display
func formatValidation(validations []Validation) []string {
	var result []string
	var minVal, maxVal string

	for _, validation := range validations {
		switch {
		case strings.HasPrefix(validation.Name, "min("):
			minVal = extractValue(validation.Name, "min(")
		case strings.HasPrefix(validation.Name, "max("):
			maxVal = extractValue(validation.Name, "max(")
		case strings.HasPrefix(validation.Name, "oneOf("):
			// Extract values from oneOf(format)
			values := extractOneOfValues(validation.Name)
			result = append(result, fmt.Sprintf("oneOf: %s", values))
		case strings.HasPrefix(validation.Name, "minLength("):
			min := extractValue(validation.Name, "minLength(")
			result = append(result, fmt.Sprintf("minLength: %s", min))
		case strings.HasPrefix(validation.Name, "maxLength("):
			max := extractValue(validation.Name, "maxLength(")
			result = append(result, fmt.Sprintf("maxLength: %s", max))
		case strings.HasPrefix(validation.Name, "regexp("):
			pattern := extractValue(validation.Name, "regexp(")
			result = append(result, fmt.Sprintf("pattern: %s", pattern))
		default:
			// For other validations, use the name as-is
			result = append(result, validation.Name)
		}
	}

	// Handle min/max range
	if minVal != "" && maxVal != "" {
		result = append([]string{fmt.Sprintf("valid: %s-%s", minVal, maxVal)}, result...)
	} else if minVal != "" {
		result = append([]string{fmt.Sprintf("min: %s", minVal)}, result...)
	} else if maxVal != "" {
		result = append([]string{fmt.Sprintf("max: %s", maxVal)}, result...)
	}

	return result
}

// extractValue extracts numeric value from validation name like "min(8080)"
func extractValue(name, prefix string) string {
	start := strings.Index(name, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(name[start:], ")")
	if end == -1 {
		return name[start:]
	}
	return name[start : start+end]
}

// extractOneOfValues extracts values from oneOf(['a', 'b', 'c']) format
func extractOneOfValues(name string) string {
	start := strings.Index(name, "oneOf(")
	if start == -1 {
		return ""
	}
	start += len("oneOf(")
	end := strings.Index(name[start:], ")")
	if end == -1 {
		return name[start:]
	}
	values := name[start : start+end]

	// Handle array format oneOf([debug info warn error])
	if strings.HasPrefix(values, "[") && strings.HasSuffix(values, "]") {
		// Remove brackets and clean up
		content := values[1 : len(values)-1]
		// Split by space and filter empty strings
		parts := strings.Fields(content)
		var quotedParts []string
		for _, part := range parts {
			if part != "" {
				quotedParts = append(quotedParts, fmt.Sprintf("'%s'", part))
			}
		}
		var sb strings.Builder
		sb.Grow(len(quotedParts)*8 + 2) // Pre-allocate estimated capacity
		sb.WriteString("[")
		sb.WriteString(strings.Join(quotedParts, ", "))
		sb.WriteString("]")
		return sb.String()
	}

	// Handle simple format oneOf(a,b,c)
	parts := strings.Split(values, ",")
	var quotedParts []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			quotedParts = append(quotedParts, fmt.Sprintf("'%s'", strings.TrimSpace(part)))
		}
	}
	var sb strings.Builder
	sb.Grow(len(quotedParts)*8 + 2) // Pre-allocate estimated capacity
	sb.WriteString("[")
	sb.WriteString(strings.Join(quotedParts, ", "))
	sb.WriteString("]")
	return sb.String()
}

// newDefinitionBuilder creates a new builder
func newDefinitionBuilder(cfg *Config, key string) *DefinitionBuilder {
	return &DefinitionBuilder{
		def: &Definition{
			key:       key,
			valueType: TypeString, // default
			delimiter: ",",        // default delimiter
		},
		config: cfg,
	}
}

// Type setters

func (b *DefinitionBuilder) String() *DefinitionBuilder {
	b.def.valueType = TypeString
	return b
}

func (b *DefinitionBuilder) Int64() *DefinitionBuilder {
	b.def.valueType = TypeInt64
	return b
}

func (b *DefinitionBuilder) Float64() *DefinitionBuilder {
	b.def.valueType = TypeFloat64
	return b
}

func (b *DefinitionBuilder) Bool() *DefinitionBuilder {
	b.def.valueType = TypeBool
	return b
}

func (b *DefinitionBuilder) Duration() *DefinitionBuilder {
	b.def.valueType = TypeDuration
	return b
}

func (b *DefinitionBuilder) StringSlice() *DefinitionBuilder {
	b.def.valueType = TypeStringSlice
	return b
}

func (b *DefinitionBuilder) Int64Slice() *DefinitionBuilder {
	b.def.valueType = TypeInt64Slice
	return b
}

// Source setters

func (b *DefinitionBuilder) Env(envVar string) *DefinitionBuilder {
	b.def.envVar = envVar
	return b
}

func (b *DefinitionBuilder) Flag(flag string) *DefinitionBuilder {
	b.def.flag = flag
	return b
}

func (b *DefinitionBuilder) File(fileKey string) *DefinitionBuilder {
	b.def.fileKey = fileKey
	return b
}

// Priority sets the custom priority order for this definition
// The priority is validated during configuration processing
func (b *DefinitionBuilder) Priority(priority SourcePriority) *DefinitionBuilder {
	b.def.priority = append(SourcePriority(nil), priority...)
	return b
}

// Behavior setters

func (b *DefinitionBuilder) Required() *DefinitionBuilder {
	b.def.required = true
	return b
}

func (b *DefinitionBuilder) Secret() *DefinitionBuilder {
	b.def.secret = true
	return b
}

func (b *DefinitionBuilder) Default(value any) *DefinitionBuilder {
	// If we know the target type, try to convert immediately for better error detection
	if b.def.valueType != TypeString && b.def.valueType != 0 {
		converter := NewTypeConverter()
		convertedValue, err := converter.ConvertDefaultValue(value, b.def.valueType)
		if err != nil {
			// Store original value but the error will be caught during processing
			// This allows for better error messages at processing time
		} else {
			// Store the converted value for immediate type correctness
			b.def.defaultValue = convertedValue
			return b
		}
	}

	// Store original value if conversion failed or type is unknown
	b.def.defaultValue = value
	return b
}

func (b *DefinitionBuilder) Delimiter(d string) *DefinitionBuilder {
	b.def.delimiter = d
	return b
}

func (b *DefinitionBuilder) Description(desc string) *DefinitionBuilder {
	b.def.description = desc
	return b
}

// Validation setters

func (b *DefinitionBuilder) Min(min float64) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMin(min))
	return b
}

func (b *DefinitionBuilder) Max(max float64) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMax(max))
	return b
}

func (b *DefinitionBuilder) Range(min, max float64) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMin(min))
	b.def.validations = append(b.def.validations, validateMax(max))
	return b
}

func (b *DefinitionBuilder) MinLength(min int) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMinLength(min))
	return b
}

func (b *DefinitionBuilder) MaxLength(max int) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMaxLength(max))
	return b
}

func (b *DefinitionBuilder) Regexp(pattern string) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateRegexp(pattern))
	return b
}

// OneOf restricts the value to one of the allowed strings (string values only)
func (b *DefinitionBuilder) OneOf(allowed ...string) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateOneOf(allowed))
	return b
}

func (b *DefinitionBuilder) MinDuration(min time.Duration) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMinDuration(min))
	return b
}

func (b *DefinitionBuilder) MaxDuration(max time.Duration) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMaxDuration(max))
	return b
}

func (b *DefinitionBuilder) MinItems(min int) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMinItems(min))
	return b
}

func (b *DefinitionBuilder) MaxItems(max int) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, validateMaxItems(max))
	return b
}

// Custom adds a custom validation function
func (b *DefinitionBuilder) Custom(name string, check func(value any) error) *DefinitionBuilder {
	b.def.validations = append(b.def.validations, Validation{Name: name, Check: check})
	return b
}

// Helper functions for priority resolution

// getEffectivePriority returns the priority to use for this definition
// Uses definition's priority if set, otherwise falls back to config default
func (d *Definition) getEffectivePriority(configDefault SourcePriority) SourcePriority {
	if len(d.priority) > 0 {
		return d.priority
	}
	return configDefault
}

// inferAvailableSources determines which sources are available for this definition
// based on the configured fields (envVar, flag, defaultValue, etc.)
func (d *Definition) inferAvailableSources() []SourceType {
	var sources []SourceType

	// Add Default source if defaultValue is set
	if d.defaultValue != nil {
		sources = append(sources, SourceDefault)
	}

	// Add Env source if envVar is set
	if d.envVar != "" {
		sources = append(sources, SourceEnv)
	}

	// Add Flag source if flag is set
	if d.flag != "" {
		sources = append(sources, SourceFlag)
	}

	// Always include File source (files can provide any key)
	sources = append(sources, SourceFile)

	return sources
}

// validatePriority checks if the priority order is valid for this definition
func (d *Definition) validatePriority(priority SourcePriority) error {
	availableSources := d.inferAvailableSources()
	availableSet := make(map[SourceType]bool)
	for _, source := range availableSources {
		availableSet[source] = true
	}

	// Check that all sources in priority are available
	for _, source := range priority {
		if !availableSet[source] {
			return fmt.Errorf("priority includes unavailable source: %s", source.String())
		}
	}

	return nil
}
