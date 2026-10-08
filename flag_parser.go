// cli/flag_parser.go
package kli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// FlagParser provides centralized flag parsing functionality
type FlagParser interface {
	// ParseCommand parses flags for command-specific configuration
	ParseCommand(args []string, defs map[string]*Definition) (*ParsedFlags, error)

	// ParseGlobal parses flags for global configuration
	ParseGlobal(args []string, defs map[string]*Definition) (*ParsedFlags, error)

	// ConvertFlagErrorsToConfigErrors converts flag parsing errors to ConfigError instances
	ConvertFlagErrorsToConfigErrors(errors []error, defs map[string]*Definition) []ConfigError
}

// ParsedFlags contains the results of flag parsing
type ParsedFlags struct {
	Values  map[string]*string // Parsed flag values
	FlagSet *flag.FlagSet      // The actual FlagSet used
	Errors  []error            // Any parsing errors encountered
}

// flagParser implements FlagParser interface
// boolFlag stores a boolean flag's value as a string, like every other
// flag, but lets the flag package treat it as a switch: --flag means true,
// --flag=false means false, and --flag never swallows the next argument.
type boolFlag string

func (b *boolFlag) String() string {
	if b == nil {
		return ""
	}
	return string(*b)
}

func (b *boolFlag) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("invalid boolean value %q", s)
	}
	*b = boolFlag(strconv.FormatBool(v))
	return nil
}

func (b *boolFlag) IsBoolFlag() bool { return true }

type flagParser struct{}

// newFlagParser creates a new FlagParser instance
func newFlagParser() FlagParser {
	return &flagParser{}
}

// ParseCommand parses flags for command-specific configuration
func (fp *flagParser) ParseCommand(args []string, defs map[string]*Definition) (*ParsedFlags, error) {
	return fp.parseFlags(args, defs, "")
}

// ParseGlobal parses flags for global configuration
func (fp *flagParser) ParseGlobal(args []string, defs map[string]*Definition) (*ParsedFlags, error) {
	// For global parsing, use the executable name as the FlagSet name
	executable := os.Args[0]
	if executable == "" {
		executable = "command"
	}

	return fp.parseFlags(args, defs, executable)
}

// parseFlags is the core flag parsing implementation
func (fp *flagParser) parseFlags(args []string, defs map[string]*Definition, flagSetName string) (*ParsedFlags, error) {
	// Create FlagSet with ContinueOnError to collect errors instead of exiting
	flagSet := flag.NewFlagSet(flagSetName, flag.ContinueOnError)

	// Suppress Go's flag package automatic output to prevent duplication
	flagSet.SetOutput(io.Discard)

	// Create values map and register flags with correct types
	values := make(map[string]*string)
	for key, def := range defs {
		if def.flag == "" {
			continue
		}
		if def.valueType == TypeBool {
			p := new(string)
			flagSet.Var((*boolFlag)(p), def.flag, def.description)
			values[key] = p
			continue
		}
		// Every other flag is a string here; the type conversion happens
		// during config processing.
		values[key] = flagSet.String(def.flag, "", def.description)
	}

	// Parse flags and collect any errors
	err := flagSet.Parse(args)

	// Create ParsedFlags result
	result := &ParsedFlags{
		Values:  values,
		FlagSet: flagSet,
	}

	// Collect parsing errors
	if err != nil {
		result.Errors = []error{err}
	}

	return result, nil
}

// ConvertFlagErrorsToConfigErrors converts flag parsing errors to ConfigError instances
func (fp *flagParser) ConvertFlagErrorsToConfigErrors(errors []error, defs map[string]*Definition) []ConfigError {
	var configErrs []ConfigError

	for _, err := range errors {
		// Try to extract the problematic flag from the error message
		errMsg := err.Error()
		var flagName string

		// Common flag error patterns
		if strings.Contains(errMsg, "flag needs an argument: -") {
			// Extract flag name from "flag needs an argument: -debug"
			parts := strings.Split(errMsg, "-")
			if len(parts) > 1 {
				flagName = parts[1]
			}
		} else if strings.Contains(errMsg, "invalid flag: -") {
			// Extract flag name from "invalid flag: -unknown"
			parts := strings.Split(errMsg, "-")
			if len(parts) > 1 {
				flagName = parts[1]
			}
		} else if strings.Contains(errMsg, "provided but not defined: -") {
			// Extract flag name from "flag provided but not defined: -unknown"
			parts := strings.Split(errMsg, "-")
			if len(parts) > 1 {
				flagName = parts[1]
			}
		}

		// Find the corresponding definition
		var def *Definition
		if flagName != "" {
			for key, d := range defs {
				if d.flag == flagName {
					def = d
					flagName = key
					break
				}
			}
		}

		// If we can't determine the specific flag, create a generic error
		if def == nil {
			// Create a generic definition for unknown flag errors
			def = &Definition{
				key:         "unknown_flag",
				flag:        flagName,
				valueType:   TypeString,
				description: "Unknown flag",
			}
			flagName = "unknown_flag"
		}

		// Create ConfigError
		configErr := newConfigError(flagName, def, "flag", "", err)
		configErrs = append(configErrs, configErr)
	}

	return configErrs
}
