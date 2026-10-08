// cli/execution_context.go
package kli

import (
	"fmt"
	"maps"
	"strings"
	"sync"
)

// ExecutionContext provides thread-safe error collection for command execution
type ExecutionContext struct {
	errors  []GetError
	command string
	mu      sync.Mutex
}

// NewExecutionContext creates a new execution context for the specified command
func NewExecutionContext(command string) *ExecutionContext {
	return &ExecutionContext{
		errors:  make([]GetError, 0),
		command: command,
	}
}

// CollectError adds an error to the execution context with thread safety
func (ctx *ExecutionContext) CollectError(c *Config, key, expectedType, actualType, message string, isSecret bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// If display is not provided, try to build it from config
	display := ""
	if c != nil {
		flag := ""
		envVar := ""
		if def, hasDef := c.definitions[key]; hasDef {
			flag = def.flag
			envVar = def.envVar
		}
		display = getErrorDisplayName(GetError{Key: key, Flag: flag, EnvVar: envVar}, c)
	}

	// Get flag and envVar from config if available
	flag := ""
	envVar := ""
	if c != nil {
		if def, hasDef := c.definitions[key]; hasDef {
			flag = def.flag
			envVar = def.envVar
		}
	}

	err := GetError{
		Key:              key,
		ExpectedType:     expectedType,
		ActualType:       actualType,
		Message:          message,
		IsSecret:         isSecret,
		Flag:             flag,
		EnvVar:           envVar,
		Display:          display,
		ErrorDescription: message,
		config:           c,
	}
	ctx.errors = append(ctx.errors, err)
}

// CollectConfigError adds a normalized config error to the execution context
// This method preserves the existing Display and ErrorDescription from ConfigError
func (ctx *ExecutionContext) CollectConfigError(c *Config, configErr ConfigError) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	flag := ""
	envVar := ""
	if c != nil {
		if def, hasDef := c.definitions[configErr.Key]; hasDef {
			flag = def.flag
			envVar = def.envVar
		}
	}

	err := GetError{
		Key:              configErr.Key,
		ExpectedType:     "validation",
		ActualType:       "",
		Message:          configErr.ErrorDescription,
		IsSecret:         false,
		Flag:             flag,
		EnvVar:           envVar,
		Display:          configErr.Display,          // Use existing display from ConfigError
		ErrorDescription: configErr.ErrorDescription, // Use existing description from ConfigError
		config:           c,
	}
	ctx.errors = append(ctx.errors, err)
}

// HasErrors returns true if there are collected errors
func (ctx *ExecutionContext) HasErrors() bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return len(ctx.errors) > 0
}

// GetErrors returns a copy of all collected errors
func (ctx *ExecutionContext) GetErrors() []GetError {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	errs := make([]GetError, len(ctx.errors))
	copy(errs, ctx.errors)
	return errs
}

func (ctx *ExecutionContext) synthesizeCommand(errs []GetError) *Command {
	definitions := make(map[string]*Definition)
	for _, err := range errs {
		if err.config == nil {
			continue
		}
		maps.Copy(definitions, err.config.definitions)
		break
	}

	commandName := ctx.GetCommand()
	if commandName == "" {
		commandName = "command"
	}

	return &Command{
		Name:        commandName,
		Definitions: definitions,
		SubCommands: make(map[string]*Command),
	}
}

func (ctx *ExecutionContext) renderErrorsWithCommand(cmd *Command, helpService *helpService) (string, error) {
	errs := ctx.GetErrors()
	if len(errs) == 0 {
		return "", nil
	}

	if helpService == nil {
		helpService = newHelpService()
	}

	if cmd == nil {
		cmd = ctx.synthesizeCommand(errs)
	}

	// Create a simple help display for errors
	var builder strings.Builder
	if exe := helpService.coordinator.executable; ctx.command == exe || ctx.command == "" {
		builder.WriteString(fmt.Sprintf("Usage: %s [options]\n\n", exe))
	} else {
		builder.WriteString(fmt.Sprintf("Usage: %s %s [options]\n\n", exe, ctx.command))
	}

	if cmd != nil && cmd.LongHelp != "" {
		builder.WriteString(cmd.LongHelp)
		builder.WriteString("\n\n")
	}

	if len(errs) > 0 {
		builder.WriteString("Configuration errors:\n")
		for _, err := range errs {
			builder.WriteString(fmt.Sprintf("  %s -> %s\n", err.Display, err.ErrorDescription))
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// GetCommand returns the command name for this context
func (ctx *ExecutionContext) GetCommand() string {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.command
}

// SetCommand updates the command name for this context
func (ctx *ExecutionContext) SetCommand(command string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.command = command
}

// Clear removes all collected errors (useful for testing)
func (ctx *ExecutionContext) Clear() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.errors = nil
}
