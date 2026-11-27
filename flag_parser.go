// cli/flag_parser.go
package kli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
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
	ConvertFlagErrorsToConfigErrors(errs []error, defs map[string]*Definition) []ConfigError
}

// ParsedFlags contains the results of flag parsing
type ParsedFlags struct {
	Values  map[string]*string // Parsed flag values
	FlagSet *flag.FlagSet      // The actual FlagSet used
	Errors  []error            // Any parsing errors encountered
}

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
	v, err := parseBool(s)
	if err != nil {
		return err
	}
	*b = boolFlag(strconv.FormatBool(v))
	return nil
}

func (b *boolFlag) IsBoolFlag() bool { return true }

// checkDuplicateFlags returns an error if two definitions use one flag name
func checkDuplicateFlags(defs map[string]*Definition) error {
	owners := make(map[string]string) // flag name -> definition key
	// Sorted so that a duplicated name is always reported the same way.
	for _, key := range slices.Sorted(maps.Keys(defs)) {
		for _, name := range defs[key].flags {
			if owner, used := owners[name]; used {
				return fmt.Errorf("flag %s duplicated on %q and %q",
					formatFlagNames([]string{name}), owner, key)
			}
			owners[name] = key
		}
	}
	return nil
}

// flagParser implements FlagParser interface
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
	// The flag package panics on a redefined name, so check first.
	if err := checkDuplicateFlags(defs); err != nil {
		return &ParsedFlags{Values: values, FlagSet: flagSet}, err
	}

	for key, def := range defs {
		if len(def.flags) == 0 {
			continue
		}
		// All the names of a definition share one value, so any of them sets
		// it and the last one on the command line wins.
		p := new(string)
		values[key] = p
		for _, name := range def.flags {
			if def.valueType == TypeBool {
				flagSet.Var((*boolFlag)(p), name, def.description)
				continue
			}
			// Every other flag is a string here; the type conversion happens
			// during config processing.
			flagSet.StringVar(p, name, "", def.description)
		}
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
func (fp *flagParser) ConvertFlagErrorsToConfigErrors(errs []error, defs map[string]*Definition) []ConfigError {
	var configErrs []ConfigError

	for _, err := range errs {
		// Try to extract the problematic flag from the error message
		errMsg := err.Error()
		var flagName string

		// Common flag error patterns, each followed by the flag name, as in
		// "flag needs an argument: -dry-run"
		for _, pattern := range []string{
			"flag needs an argument: -",
			"invalid flag: -",
			"provided but not defined: -",
		} {
			if _, name, found := strings.Cut(errMsg, pattern); found {
				flagName = name
				break
			}
		}

		// A rejected boolean reads `invalid boolean value "x" for -name: reason`.
		// Only the reason is kept, so it matches the one for other sources.
		if rest, found := strings.CutPrefix(errMsg, "invalid boolean value "); found {
			if value, quoteErr := strconv.QuotedPrefix(rest); quoteErr == nil {
				rest, found = strings.CutPrefix(rest[len(value):], " for -")
				if name, reason, ok := strings.Cut(rest, ": "); found && ok {
					flagName, err = name, errors.New(reason)
				}
			}
		}

		// Find the corresponding definition
		var def *Definition
		if flagName != "" {
			for key, d := range defs {
				if slices.Contains(d.flags, flagName) {
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
				valueType:   TypeString,
				description: "Unknown flag",
			}
			if flagName != "" {
				def.flags = []string{flagName}
			}
			flagName = "unknown_flag"
		}

		// Create ConfigError
		configErr := newConfigError(flagName, def, "flag", "", err)
		configErrs = append(configErrs, configErr)
	}

	return configErrs
}
