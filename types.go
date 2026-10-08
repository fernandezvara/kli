// cli/types.go
package kli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ValueType represents the expected type of a configuration value
type ValueType int

const (
	TypeString ValueType = iota
	TypeInt64
	TypeFloat64
	TypeBool
	TypeDuration
	TypeStringSlice
	TypeInt64Slice
)

func (t ValueType) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeInt64:
		return "int64"
	case TypeFloat64:
		return "float64"
	case TypeBool:
		return "bool"
	case TypeDuration:
		return "duration"
	case TypeStringSlice:
		return "[]string"
	case TypeInt64Slice:
		return "[]int64"
	default:
		return "unknown"
	}
}

// SourceType represents a configuration source type
type SourceType int

const (
	SourceDefault SourceType = iota
	SourceFlag
	SourceEnv
	SourceFile
)

func (s SourceType) String() string {
	switch s {
	case SourceDefault:
		return "default"
	case SourceFlag:
		return "flag"
	case SourceEnv:
		return "environment"
	case SourceFile:
		return "file"
	default:
		return "unknown"
	}
}

// SourcePriority defines the priority order for configuration sources
type SourcePriority []SourceType

// Common priority presets
var (
	// PriorityFlagEnvFileDefault sets Flag > Env > File > Default priority (current default)
	PriorityFlagEnvFileDefault = SourcePriority{SourceFlag, SourceEnv, SourceFile, SourceDefault}

	// PriorityFlagEnvDefault sets Flag > Env > Default priority
	PriorityFlagEnvDefault = SourcePriority{SourceFlag, SourceEnv, SourceDefault}

	// PriorityEnvFlagDefault sets Env > Flag > Default priority
	PriorityEnvFlagDefault = SourcePriority{SourceEnv, SourceFlag, SourceDefault}

	// PriorityFileEnvFlagDefault sets File > Env > Flag > Default priority
	PriorityFileEnvFlagDefault = SourcePriority{SourceFile, SourceEnv, SourceFlag, SourceDefault}

	// PriorityDefaultOnly uses only Default values
	PriorityDefaultOnly = SourcePriority{SourceDefault}
)

// parseValue parses a string value into the expected type
func parseValue(raw string, valueType ValueType, delimiter string) (any, error) {
	if raw == "" {
		return nil, nil
	}

	switch valueType {
	case TypeString:
		return raw, nil

	case TypeInt64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid int64: %s", raw)
		}
		return v, nil

	case TypeFloat64:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float64: %s", raw)
		}
		return v, nil

	case TypeBool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bool: %s (use true/false, 1/0, yes/no)", raw)
		}
		return v, nil

	case TypeDuration:
		// Handle day format (e.g., "7d", "1d")
		if before, ok := strings.CutSuffix(raw, "d"); ok {
			daysStr := before
			if days, err := strconv.ParseFloat(daysStr, 64); err == nil {
				hours := days * 24
				v, err := time.ParseDuration(fmt.Sprintf("%.0fh", hours))
				if err != nil {
					return nil, fmt.Errorf("invalid duration: %s", raw)
				}
				return v, nil
			}
		}
		v, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %s (use format like 15m, 1h, 7d)", raw)
		}
		return v, nil

	case TypeStringSlice:
		parts := strings.Split(raw, delimiter)
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result, nil

	case TypeInt64Slice:
		parts := strings.Split(raw, delimiter)
		result := make([]int64, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				continue
			}
			v, err := strconv.ParseInt(trimmed, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid int64 in array: %s", trimmed)
			}
			result = append(result, v)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown type: %v", valueType)
	}
}
