// cli/command_router.go
package kli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// CommandRouter routes commands and handles subcommands
type CommandRouter interface {
	// HandleSubcommands checks for and handles subcommand routing
	HandleSubcommands(cmd *Command, ctx *CommandContext) (*Command, *CommandContext, error)

	// RouteWithHelpHandling integrates help detection with command routing
	RouteWithHelpHandling(args []string, config *Config) (*Command, *CommandContext, error)
}

// commandRouter implements CommandRouter interface
type commandRouter struct{}

// newCommandRouter creates a new CommandRouter instance
func newCommandRouter() CommandRouter {
	return &commandRouter{}
}

// HandleSubcommands checks for and handles subcommand routing
func (cr *commandRouter) HandleSubcommands(cmd *Command, ctx *CommandContext) (*Command, *CommandContext, error) {
	if cmd == nil {
		return nil, ctx, fmt.Errorf("command cannot be nil")
	}

	if ctx == nil {
		return nil, nil, fmt.Errorf("context cannot be nil")
	}

	// Check for subcommands
	if len(ctx.Args) > 0 {
		subCmdName := ctx.Args[0]
		if subCmd := cmd.FindSubCommand(subCmdName); subCmd != nil {
			// Update context for subcommand
			ctx.SubCommand = subCmdName
			ctx.Args = ctx.Args[1:]

			// Update execution context to include subcommand
			if ctx.execution != nil {
				ctx.execution.SetCommand(ctx.Command + " " + subCmdName)
			}

			return subCmd, ctx, nil
		}
	}

	return cmd, ctx, nil
}

// findCommand finds a top-level command by name or alias
func findCommand(commands map[string]*Command, name string) *Command {
	if cmd, exists := commands[name]; exists {
		return cmd
	}

	for _, cmd := range commands {
		if slices.Contains(cmd.Aliases, name) {
			return cmd
		}
	}

	return nil
}

// RouteWithHelpHandling integrates help detection with command routing
func (cr *commandRouter) RouteWithHelpHandling(args []string, config *Config) (*Command, *CommandContext, error) {
	if config == nil {
		return nil, nil, fmt.Errorf("config cannot be nil")
	}

	// Handle no command case - check for empty string default command
	if len(args) < 2 {
		if defaultCmd, exists := config.commands[""]; exists {
			// Route to empty string command with all args as flags
			// For empty string command, all args after program name are flags
			var flagArgs []string
			if len(args) > 1 {
				flagArgs = args[1:] // All args after program name are flags
			}
			ctx := NewCommandContext(flagArgs, config, "", "")
			return defaultCmd, ctx, nil
		}
		// Fall back to global help if no default command
		err := config.getHelpService().ShowHelpUnified("", "", false, []GetError{}, config.commands)
		return nil, nil, err // Help shown, no command to execute
	}

	// Extract command name and remaining args
	commandName := args[1]
	remainingArgs := args[2:]

	// Check if the command name is actually a help flag
	if commandName == "help" && findCommand(config.commands, "help") == nil {
		commandName = "--help"
	}

	// A flag in place of the command name belongs to the default command
	isFlag := strings.HasPrefix(commandName, "-")
	defaultCmd, hasDefault := config.commands[""]
	requested, full := helpRequest(args[1:], config.definitions)

	// Help for the command named after the flags, if any, as in
	// "app help deploy" or "app --help deploy"
	if isHelpFlag(commandName) || (isFlag && requested && !hasDefault) {
		topics := helpTopics(args[1:], config.definitions)
		return nil, nil, showHelpForTopics(config, topics, full)
	}

	// Check if this should route to empty string command (when args[1] looks like a flag)
	if isFlag && hasDefault {
		// All args are flags of the default command
		ctx := NewCommandContext(args[1:], config, "", "")
		if requested, full := helpRequest(ctx.Args, definitionsFor(defaultCmd, config)); requested {
			return nil, nil, config.getHelpService().ShowHelpUnified("", "", full, []GetError{}, config.commands)
		}
		return defaultCmd, ctx, nil
	}

	// Find command by name or alias
	cmd := findCommand(config.commands, commandName)
	if cmd == nil {
		msg := fmt.Sprintf("unknown command: %q", commandName)
		if suggestions := config.findSuggestions(commandName); suggestions != "no similar commands found" {
			msg += fmt.Sprintf("\nDid you mean: %s?", suggestions)
		}
		return nil, nil, usageError(errors.New(msg), false)
	}

	// Create initial command context
	ctx := NewCommandContext(remainingArgs, config, cmd.Name, "")

	// Handle subcommands first to get the full command path
	finalCmd, finalCtx, err := cr.HandleSubcommands(cmd, ctx)
	if err != nil {
		return nil, nil, err
	}

	// Check for help requests using centralized detection
	if len(finalCtx.Args) == 1 && finalCtx.Args[0] == "help" && finalCmd.FindSubCommand("help") == nil {
		finalCtx.Args = []string{"--help"}
	}

	defs := definitionsFor(finalCmd, config)
	if requested, full := helpRequest(finalCtx.Args, defs); requested {
		subCommand := finalCtx.SubCommand
		// The subcommand may come after the flag, as in "app user -h list"
		if topics := helpTopics(finalCtx.Args, defs); subCommand == "" && len(topics) > 0 {
			if subCmd := finalCmd.FindSubCommand(topics[0]); subCmd != nil {
				subCommand = subCmd.Name
			}
		}
		err := config.getHelpService().ShowHelpUnified(finalCtx.Command, subCommand, full, []GetError{}, config.commands)
		return nil, nil, err // Help shown, no command to execute
	}

	return finalCmd, finalCtx, nil
}

// definitionsFor returns the definitions a command's flags are parsed with: its
// own, or the global ones if it declares none
func definitionsFor(cmd *Command, config *Config) map[string]*Definition {
	if len(cmd.Definitions) > 0 {
		return cmd.Definitions
	}
	return config.definitions
}

// showHelpForTopics shows help for the command, and subcommand, that topics
// name. Without topics it shows the general help.
func showHelpForTopics(config *Config, topics []string, full bool) error {
	command, subCommand := "", ""
	if len(topics) > 0 {
		command = topics[0]
		if cmd := findCommand(config.commands, command); cmd != nil {
			command = cmd.Name
			if len(topics) > 1 {
				if subCmd := cmd.FindSubCommand(topics[1]); subCmd != nil {
					subCommand = subCmd.Name
				}
			}
		}
	}
	return config.getHelpService().ShowHelpUnified(command, subCommand, full, []GetError{}, config.commands)
}
