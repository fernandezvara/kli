// cli/type_converter.go
package kli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// TypeConverter handles type-to-string conversions consistently across all value sources
type TypeConverter struct{}

// NewTypeConverter creates a new TypeConverter instance
func NewTypeConverter() *TypeConverter {
	return &TypeConverter{}
}

// ConvertToString converts any value to string representation for processing
// Returns error for unsupported types to ensure proper error handling
func (tc *TypeConverter) ConvertToString(value any, delimiter string) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", v), nil
	case time.Duration:
		return v.String(), nil
	case []string:
		return strings.Join(v, delimiter), nil
	case []int64:
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = fmt.Sprintf("%d", item)
		}
		return strings.Join(strs, delimiter), nil
	case []any:
		// Handle arrays from files - convert to strings and join
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(strs, delimiter), nil
	default:
		return "", fmt.Errorf("unsupported value type: %T", v)
	}
}

// ConvertToDisplayString converts value to string for display purposes
// Falls back to fmt.Sprintf for unsupported types to ensure display always works
func (tc *TypeConverter) ConvertToDisplayString(value any, delimiter string) string {
	str, err := tc.ConvertToString(value, delimiter)
	if err != nil {
		// For display purposes, fall back to basic string conversion
		return fmt.Sprintf("%v", value)
	}
	return str
}

// IsSupportedType checks if a type is supported for conversion
func (tc *TypeConverter) IsSupportedType(value any) bool {
	switch value.(type) {
	case string, bool, int, int64, float64, time.Duration:
		return true
	case []string, []int64, []any:
		return true
	default:
		return false
	}
}

// convertDefaultValue handles type conversion for default values based on target ValueType
func convertDefaultValue(value any, targetType ValueType) (any, error) {
	if value == nil {
		return nil, nil
	}

	// If value is already the correct type, return it directly
	switch targetType {
	case TypeString:
		if _, ok := value.(string); ok {
			return value, nil
		}
	case TypeInt64:
		if _, ok := value.(int64); ok {
			return value, nil
		}
	case TypeFloat64:
		if _, ok := value.(float64); ok {
			return value, nil
		}
	case TypeBool:
		if _, ok := value.(bool); ok {
			return value, nil
		}
	case TypeDuration:
		if _, ok := value.(time.Duration); ok {
			return value, nil
		}
	}

	// Perform type conversions based on target type
	switch targetType {
	case TypeString:
		return fmt.Sprintf("%v", value), nil

	case TypeInt64:
		switch v := value.(type) {
		case int, int8, int16, int32:
			return int64(reflect.ValueOf(v).Int()), nil
		case uint, uint8, uint16, uint32:
			return int64(reflect.ValueOf(v).Uint()), nil
		case uint64:
			if v > uint64(1<<63-1) {
				return nil, fmt.Errorf("value %d overflows int64", v)
			}
			return int64(v), nil
		case float64, float32:
			f := reflect.ValueOf(v).Float()
			if f > float64(1<<63-1) || f < float64(-1<<63) {
				return nil, fmt.Errorf("value %f overflows int64", f)
			}
			return int64(f), nil
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert string '%s' to int64: %w", v, err)
			}
			return parsed, nil
		case bool:
			if v {
				return int64(1), nil
			}
			return int64(0), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to int64", value)
		}

	case TypeFloat64:
		switch v := value.(type) {
		case int, int8, int16, int32, int64:
			return float64(reflect.ValueOf(v).Int()), nil
		case uint, uint8, uint16, uint32, uint64:
			return float64(reflect.ValueOf(v).Uint()), nil
		case float32:
			return float64(v), nil
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert string '%s' to float64: %w", v, err)
			}
			return parsed, nil
		case bool:
			if v {
				return float64(1), nil
			}
			return float64(0), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to float64", value)
		}

	case TypeBool:
		switch v := value.(type) {
		case string:
			lower := strings.ToLower(v)
			switch lower {
			case "true", "1", "yes", "on", "enabled":
				return true, nil
			case "false", "0", "no", "off", "disabled":
				return false, nil
			default:
				return nil, fmt.Errorf("cannot convert string '%s' to bool", v)
			}
		case int, int8, int16, int32, int64:
			return reflect.ValueOf(v).Int() != 0, nil
		case uint, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(v).Uint() != 0, nil
		case float32, float64:
			return reflect.ValueOf(v).Float() != 0, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to bool", value)
		}

	case TypeDuration:
		switch v := value.(type) {
		case string:
			parsed, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("cannot convert string '%s' to duration: %w", v, err)
			}
			return parsed, nil
		case int, int8, int16, int32, int64:
			// Assume seconds for integer values
			seconds := reflect.ValueOf(v).Int()
			return time.Duration(seconds) * time.Second, nil
		case float32, float64:
			// Assume seconds for float values
			seconds := reflect.ValueOf(v).Float()
			return time.Duration(seconds) * time.Second, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to duration", value)
		}

	case TypeStringSlice:
		switch v := value.(type) {
		case []string:
			return v, nil
		case []int64:
			result := make([]string, len(v))
			for i, item := range v {
				result[i] = fmt.Sprintf("%d", item)
			}
			return result, nil
		case []any:
			// Arrays loaded from config files
			result := make([]string, len(v))
			for i, item := range v {
				result[i] = fmt.Sprintf("%v", item)
			}
			return result, nil
		case string:
			return []string{v}, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to []string", value)
		}

	case TypeInt64Slice:
		switch v := value.(type) {
		case []int64:
			return v, nil
		case []string:
			result := make([]int64, len(v))
			for i, item := range v {
				parsed, err := strconv.ParseInt(item, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("cannot convert string '%s' to int64 in slice: %w", item, err)
				}
				result[i] = parsed
			}
			return result, nil
		case []any:
			// Arrays loaded from config files
			result := make([]int64, len(v))
			for i, item := range v {
				switch n := item.(type) {
				case int64:
					result[i] = n
				case float64:
					result[i] = int64(n)
				case int:
					result[i] = int64(n)
				default:
					return nil, fmt.Errorf("cannot convert %T to int64 in slice", item)
				}
			}
			return result, nil
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot convert string '%s' to int64: %w", v, err)
			}
			return []int64{parsed}, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to []int64", value)
		}

	default:
		return nil, fmt.Errorf("cannot convert %T to %s", value, targetType)
	}
}

// ConvertDefaultValue converts a default value to the target ValueType
// This is the main method that should be used for default value type conversion
func (tc *TypeConverter) ConvertDefaultValue(value any, targetType ValueType) (any, error) {
	return convertDefaultValue(value, targetType)
}
