// cli/command_context.go
package kli

import "io"

// CommandContext provides context for command execution
type CommandContext struct {
	Args          []string
	GlobalConfig  *Config // Immutable global config
	CommandConfig *Config // Immutable command-specific config (nil if no command defs)
	Command       string
	SubCommand    string
	positional    []string          // non-flag arguments left after parsing the command's flags
	data          map[string]any    // For middleware data sharing
	execution     *ExecutionContext // Thread-safe error collection
}

// NewCommandContext creates a new command context
func NewCommandContext(args []string, config *Config, command, subCommand string) *CommandContext {
	return &CommandContext{
		Args:          args,
		GlobalConfig:  config,
		CommandConfig: nil, // Will be set by ConfigProcessor if command has definitions
		Command:       command,
		SubCommand:    subCommand,
		data:          make(map[string]any),
		execution:     NewExecutionContext(command), // Always initialize execution context
	}
}

// Set stores data in the context for middleware sharing
// Positional returns the arguments left over once the command's flags are
// parsed (everything from the first non-flag argument on). Commands that
// take only flags use it to reject stray arguments.
func (ctx *CommandContext) Positional() []string { return ctx.positional }

// Getenv looks a variable up through the environment the Config was given
// (SetEnv), os.Getenv by default.
func (ctx *CommandContext) Getenv(name string) string { return ctx.GlobalConfig.getenv(name) }

// Stdin, Stdout and Stderr are the streams the Config was given (SetIO),
// the process's own by default. Commands should use them, not os.Std*.
func (ctx *CommandContext) Stdin() io.Reader  { return ctx.GlobalConfig.in() }
func (ctx *CommandContext) Stdout() io.Writer { return ctx.GlobalConfig.out() }
func (ctx *CommandContext) Stderr() io.Writer { return ctx.GlobalConfig.errOut() }

func (ctx *CommandContext) Set(key string, value any) {
	if ctx.data == nil {
		ctx.data = make(map[string]any)
	}
	ctx.data[key] = value
}

// GetData retrieves data from the context
func (ctx *CommandContext) GetData(key string) (any, bool) {
	if ctx.data == nil {
		return nil, false
	}
	value, exists := ctx.data[key]
	return value, exists
}

// IsHelpRequested checks if help is being requested in the current context
// This is context-specific (no args parameter) and used by config processing to skip validation when help is shown
func (ctx *CommandContext) IsHelpRequested() bool {
	return argsContainHelpFlag(ctx.Args)
}
