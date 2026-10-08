// cli/files.go
package kli

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// FileConfig represents configuration loaded from files
type FileConfig struct {
	data map[string]any
}

// LoadFile loads configuration from a single file
func (c *Config) LoadFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config map[string]any
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".toml":
		err = toml.Unmarshal(data, &config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to parse %s file: %w", ext, err)
	}

	// Store file data for resolution
	if c.fileConfig == nil {
		c.fileConfig = &FileConfig{
			data: make(map[string]any),
		}
	}

	// Merge with existing file data
	c.mergeFileData(config)

	return nil
}

// LoadFiles loads configuration from multiple files (later files override earlier ones)
func (c *Config) LoadFiles(filenames ...string) error {
	for _, filename := range filenames {
		if err := c.LoadFile(filename); err != nil {
			return fmt.Errorf("error loading %s: %w", filename, err)
		}
	}
	return nil
}

// LoadFileFromEnv loads configuration file from environment variable containing the file path
func (c *Config) LoadFileFromEnv(envVar string) error {
	filename := c.getenv(envVar)
	if filename == "" {
		return nil // No environment variable set
	}
	return c.LoadFile(filename)
}

// mergeFileData merges new config data with existing file data
func (c *Config) mergeFileData(newData map[string]any) {
	if c.fileConfig == nil {
		c.fileConfig = &FileConfig{
			data: make(map[string]any),
		}
	}

	// Simple merge - new data overrides old data
	maps.Copy(c.fileConfig.data, newData)
}

// getFileValue gets a value from file configuration using fileKey or fallback to definition key
func (c *Config) getFileValue(key string, def *Definition) (any, bool) {
	if c.fileConfig == nil {
		return nil, false
	}

	// Use fileKey if specified, otherwise use the definition key
	searchKey := key
	if def != nil && def.fileKey != "" {
		searchKey = def.fileKey
	}

	// Check for regular value (case-insensitive)
	if value, exists := c.fileConfig.data[searchKey]; exists {
		return value, true
	}

	// Check lowercase version
	if value, exists := c.fileConfig.data[strings.ToLower(searchKey)]; exists {
		return value, true
	}

	return nil, false
}

// getValueFromSource gets value from a specific source type
func (c *Config) getValueFromSource(key string, def *Definition, sourceType SourceType) (any, bool) {
	switch sourceType {
	case SourceFile:
		if fileValue, exists := c.getFileValue(key, def); exists {
			return fileValue, true
		}
		return nil, false

	case SourceEnv:
		if def.envVar != "" {
			if envVal := c.getenv(def.envVar); envVal != "" {
				return envVal, true
			}
		}
		return nil, false

	case SourceFlag:
		if def.flag != "" {
			if flagVal, ok := c.flagValues[key]; ok && flagVal != nil && *flagVal != "" {
				return *flagVal, true
			}
		}
		return nil, false

	case SourceDefault:
		if def.defaultValue != nil {
			return def.defaultValue, true
		}
		return nil, false

	default:
		return nil, false
	}
}

// resolveValueWithPriority resolves a configuration value using the specified priority order
func (c *Config) resolveValueWithPriority(key string, def *Definition) (any, SourceType, error) {
	return c.resolveValueWithPriorityContext(key, def, nil)
}

// resolveValueWithPriorityContext resolves a configuration value using the specified priority order with context awareness
func (c *Config) resolveValueWithPriorityContext(key string, def *Definition, ctx *CommandContext) (any, SourceType, error) {
	// Get the effective priority for this definition
	priority := def.getEffectivePriority(c.defaultPriority)

	// Check sources in priority order
	for _, sourceType := range priority {
		if value, exists := c.getValueFromSource(key, def, sourceType); exists {
			// Handle special case for Default source - use type conversion
			if sourceType == SourceDefault {
				// Convert default value to target type
				converter := NewTypeConverter()
				convertedValue, err := converter.ConvertDefaultValue(value, def.valueType)
				if err != nil {
					return value, sourceType, err
				}

				// Skip validation if help is requested
				if ctx != nil && ctx.IsHelpRequested() {
					return convertedValue, sourceType, nil
				}

				// Run validations (except required check for defaults)
				for _, v := range def.validations {
					if err := v.Check(convertedValue); err != nil {
						return convertedValue, sourceType, err
					}
				}
				return convertedValue, sourceType, nil
			}

			// For non-default sources, convert to string and parse using TypeConverter
			converter := NewTypeConverter()
			rawValue, err := converter.ConvertToString(value, def.delimiter)
			if err != nil {
				return value, sourceType, err
			}

			// Parse the raw string value into the expected type
			parsedValue, err := parseValue(rawValue, def.valueType, def.delimiter)
			if err != nil {
				return rawValue, sourceType, err
			}

			// Skip validation if help is requested
			if ctx != nil && ctx.IsHelpRequested() {
				return parsedValue, sourceType, nil
			}

			// Run validations
			for _, validation := range def.validations {
				if err := validation.Check(parsedValue); err != nil {
					return parsedValue, sourceType, err
				}
			}

			return parsedValue, sourceType, nil
		}
	}

	// No value found
	if def.required {
		return nil, SourceDefault, fmt.Errorf("Not provided")
	}

	return nil, SourceDefault, nil
}
