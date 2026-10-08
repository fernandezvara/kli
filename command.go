// cli/command.go
package kli

import "slices"

// Command represents a CLI command with its configuration
type Command struct {
	Name        string
	Func        CommandFunc
	ShortHelp   string
	LongHelp    string
	Aliases     []string
	Definitions map[string]*Definition
	SubCommands map[string]*Command
	Middleware  []CommandMiddleware
}

// CommandFunc represents the function that executes a command
type CommandFunc func(*CommandContext) error

// CommandMiddleware represents middleware that can wrap command execution
type CommandMiddleware func(next CommandFunc) CommandFunc

// Execute executes the command with the given context and returns CommandResult for unified error handling
func (cmd *Command) Execute(ctx *CommandContext) *CommandResult {
	// Create services and delegate to CommandExecutor
	services := newCommandServices()
	executor := services.Executor
	return executor.Execute(cmd, ctx, services)
}

// FindSubCommand finds a subcommand by name or alias
func (cmd *Command) FindSubCommand(name string) *Command {
	// Check exact name first
	if subCmd, exists := cmd.SubCommands[name]; exists {
		return subCmd
	}

	// Check aliases
	for _, subCmd := range cmd.SubCommands {
		if slices.Contains(subCmd.Aliases, name) {
			return subCmd
		}
	}

	return nil
}
