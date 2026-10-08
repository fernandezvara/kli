// cli/command_executor.go
package kli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

// CommandExecutor orchestrates command execution flow
type CommandExecutor interface {
	// Execute orchestrates the complete command execution using all services
	Execute(cmd *Command, ctx *CommandContext, services *CommandServices) *CommandResult
}

// commandExecutor implements CommandExecutor interface
type commandExecutor struct{}

// newCommandExecutor creates a new CommandExecutor instance
func newCommandExecutor() CommandExecutor {
	return &commandExecutor{}
}

// Execute orchestrates the complete command execution using all services
func (ce *commandExecutor) Execute(cmd *Command, ctx *CommandContext, services *CommandServices) *CommandResult {
	if cmd == nil {
		return errorResult(fmt.Errorf("command cannot be nil"))
	}

	if ctx == nil {
		return errorResult(fmt.Errorf("context cannot be nil"))
	}

	if services == nil {
		return errorResult(fmt.Errorf("services cannot be nil"))
	}

	// Ensure execution context exists
	if ctx.execution == nil {
		ctx.execution = NewExecutionContext(ctx.Command)
	}

	// 1. Validate command state
	if err := ce.validateCommand(cmd, ctx); err != nil {
		return err
	}

	// 2. Process configuration if needed
	if err := ce.processConfiguration(cmd, ctx, services); err != nil {
		return err
	}

	// 3. Apply and execute middleware
	return ce.executeWithMiddleware(cmd, ctx, services)
}

// validateCommand validates the command state and returns appropriate help for subcommands
func (ce *commandExecutor) validateCommand(cmd *Command, ctx *CommandContext) *CommandResult {
	// Check if command has no function but has subcommands
	if cmd.Func == nil && len(cmd.SubCommands) > 0 {
		// Use the new help system to show subcommand help
		// Get commands from the context's global config
		var commands map[string]*Command
		if ctx.GlobalConfig != nil {
			commands = ctx.GlobalConfig.commands
		} else if ctx.CommandConfig != nil {
			commands = ctx.CommandConfig.commands
		} else {
			// Fallback: create a minimal commands map with just the current command
			commands = make(map[string]*Command)
			if cmd != nil {
				commands[ctx.Command] = cmd
			}
		}

		// A word that names no subcommand is a usage error, not a request for help.
		if len(ctx.Args) > 0 && !strings.HasPrefix(ctx.Args[0], "-") {
			return &CommandResult{
				Error:    usageError(fmt.Errorf("unknown command %q for %q", ctx.Args[0], ctx.Command), false),
				ExitCode: ExitUsage,
			}
		}

		// Show subcommand help using unified system
		helpService := newHelpService()
		if ctx.GlobalConfig != nil {
			helpService = ctx.GlobalConfig.getHelpService()
		}
		err := helpService.ShowHelpUnified(ctx.Command, ctx.SubCommand, false, []GetError{}, commands)
		if err != nil {
			return errorResult(err)
		}
		return success() // Help was shown successfully
	}

	if cmd.Func == nil {
		return errorResult(fmt.Errorf("command '%s' has no implementation", ctx.Command))
	}

	return nil // Continue with execution
}

// processConfiguration handles command-specific configuration processing
func (ce *commandExecutor) processConfiguration(cmd *Command, ctx *CommandContext, services *CommandServices) *CommandResult {
	// Process command-specific configuration if any
	if len(cmd.Definitions) > 0 {
		configProcessor := services.ConfigProcessor

		// Process configuration
		if result := configProcessor.ProcessCommandConfig(cmd, ctx); result.Error != nil {
			return result // Errors already collected in ctx.execution
		}

		// Validate required flags and log warnings for designers
		if result := configProcessor.ValidateRequiredFlags(cmd, ctx); result.Error != nil {
			return result // Should not error for validation, but check anyway
		}
	} else {
		// A command that declares no flags of its own takes only the global
		// ones; whatever is left over is positional.
		defs := map[string]*Definition{}
		if ctx.GlobalConfig != nil {
			defs = ctx.GlobalConfig.definitions
		}
		flagParser := services.FlagParser
		parsed, _ := flagParser.ParseCommand(ctx.Args, defs)
		ctx.positional = parsed.FlagSet.Args()
		var errs []error
		for _, err := range parsed.Errors {
			if !errors.Is(err, flag.ErrHelp) {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			for _, configErr := range flagParser.ConvertFlagErrorsToConfigErrors(errs, defs) {
				ctx.execution.CollectConfigError(ctx.GlobalConfig, configErr)
			}
			return configErrorResult("configuration errors detected")
		}
	}

	return nil // Continue with execution
}

// executeWithMiddleware applies middleware and executes the command
func (ce *commandExecutor) executeWithMiddleware(cmd *Command, ctx *CommandContext, services *CommandServices) *CommandResult {
	middlewareChain := services.MiddlewareChain

	// Check for configuration errors BEFORE executing the command
	if ctx.execution != nil && ctx.execution.HasErrors() {
		// Return configuration error result that will trigger proper error display
		return configErrorResult("configuration errors detected")
	}

	// Apply middleware using MiddlewareChain service
	finalFunc := middlewareChain.ApplyCommandOnly(cmd, cmd.Func)

	// Execute the command function
	err := finalFunc(ctx)

	// Check for collected errors and create appropriate result
	if ctx.execution.HasErrors() {
		if err != nil {
			return errorResult(err)
		}
		return errorResult(fmt.Errorf("command execution failed with collected errors"))
	}

	// Return success or error result
	if err != nil {
		return errorResult(err)
	}

	return success()
}
