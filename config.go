// cli/config.go
package kli

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Config holds configuration definitions and values
type Config struct {
	definitions      map[string]*Definition
	values           map[string]any
	secrets          *SecretStore
	flagValues       map[string]*string
	fileConfig       *FileConfig
	commands         map[string]*Command
	globalMiddleware []CommandMiddleware
	processed        bool
	helpService      *helpService
	defaultPriority  SourcePriority // Fallback priority for definitions without explicit priority

	env    func(string) string // environment lookup; nil means os.Getenv
	stdin  io.Reader           // nil means os.Stdin
	stdout io.Writer           // nil means os.Stdout
	stderr io.Writer           // nil means os.Stderr
	name   string              // program name in help; "" means the executable's name
}

// New creates a new Config instance
func New() *Config {
	return &Config{
		definitions:      make(map[string]*Definition),
		values:           make(map[string]any),
		secrets:          newSecretStore(),
		flagValues:       make(map[string]*string),
		fileConfig:       nil,
		commands:         make(map[string]*Command),
		globalMiddleware: make([]CommandMiddleware, 0),
		processed:        false,
		defaultPriority:  PriorityFlagEnvFileDefault, // Flag > Env > File > Default
	}
}

// SetEnv replaces the environment lookup (default os.Getenv), so programs
// and tests can supply their own environment.
func (c *Config) SetEnv(getenv func(string) string) *Config {
	c.env = getenv
	return c
}

// SetIO replaces stdin, stdout and stderr (defaults os.Stdin, os.Stdout,
// os.Stderr). Help goes to stdout; configuration errors go to stderr. A nil
// argument keeps the default for that stream.
func (c *Config) SetIO(stdin io.Reader, stdout, stderr io.Writer) *Config {
	c.stdin, c.stdout, c.stderr = stdin, stdout, stderr
	if c.helpService != nil {
		c.helpService = nil // rebuilt on next use, writing to the new stdout
	}
	return c
}

// SetName sets the program name shown in usage lines (default: the
// executable's base name).
func (c *Config) SetName(name string) *Config {
	c.name = name
	if c.helpService != nil {
		c.helpService = nil
	}
	return c
}

func (c *Config) getenv(name string) string {
	if c != nil && c.env != nil {
		return c.env(name)
	}
	return os.Getenv(name)
}

func (c *Config) errOut() io.Writer {
	if c != nil && c.stderr != nil {
		return c.stderr
	}
	return os.Stderr
}

func (c *Config) in() io.Reader {
	if c != nil && c.stdin != nil {
		return c.stdin
	}
	return os.Stdin
}

func (c *Config) out() io.Writer {
	if c != nil && c.stdout != nil {
		return c.stdout
	}
	return os.Stdout
}

// Define starts a new configuration definition
func (c *Config) Define(key string) *DefinitionBuilder {
	builder := newDefinitionBuilder(c, key)
	c.definitions[key] = builder.def
	return builder
}

// Command starts a new command definition
func (c *Config) Command(name string) *CommandBuilder {
	builder := newCommandBuilder(c, name)
	c.commands[name] = builder.cmd
	return builder
}

// UseMiddleware adds global middleware that applies to all commands
func (c *Config) UseMiddleware(middleware CommandMiddleware) {
	c.globalMiddleware = append(c.globalMiddleware, middleware)
}

// UseMiddlewareForCommands adds middleware only for specific commands
func (c *Config) UseMiddlewareForCommands(commandNames []string, middleware CommandMiddleware) {
	// Create a wrapper middleware that only executes for specified commands
	wrapper := func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			if slices.Contains(commandNames, ctx.Command) {
				return middleware(next)(ctx)
			}
			return next(ctx)
		}
	}
	c.globalMiddleware = append(c.globalMiddleware, wrapper)
}

// UseMiddlewareForSubcommands adds middleware only for specific subcommands of a command
func (c *Config) UseMiddlewareForSubcommands(commandName string, subcommandNames []string, middleware CommandMiddleware) {
	// Create a wrapper middleware that only executes for specified subcommands
	wrapper := func(next CommandFunc) CommandFunc {
		return func(ctx *CommandContext) error {
			if ctx.Command == commandName {
				if slices.Contains(subcommandNames, ctx.SubCommand) {
					return middleware(next)(ctx)
				}
			}
			return next(ctx)
		}
	}
	c.globalMiddleware = append(c.globalMiddleware, wrapper)
}

// SetDefaultPriority sets the default priority order for all definitions
// that don't have an explicit priority set
func (c *Config) SetDefaultPriority(priority SourcePriority) *Config {
	c.defaultPriority = append(SourcePriority(nil), priority...)
	return c
}

// GetDefaultPriority returns the current default priority order
func (c *Config) GetDefaultPriority() SourcePriority {
	return append(SourcePriority(nil), c.defaultPriority...)
}

// processDefinitions resolves and validates all definitions, storing values into the Config.
// It assumes c.flagValues is already populated. Returns any ConfigErrors found.
func (c *Config) processDefinitions() []ConfigError {
	return c.processDefinitionsWithContext(nil)
}

// processDefinitionsWithContext resolves and validates all definitions with context awareness.
// When help is requested, validation is skipped to allow help display.
func (c *Config) processDefinitionsWithContext(ctx *CommandContext) []ConfigError {
	var errs []ConfigError

	for key, def := range c.definitions {
		// Validate custom priority orders before resolution
		if len(def.priority) > 0 {
			if err := def.validatePriority(def.priority); err != nil {
				errs = append(errs, ConfigError{
					Key:              key,
					Source:           "config",
					Display:          buildDefinitionDisplay(def),
					ErrorDescription: err.Error(),
				})
				continue
			}
		}

		var value any
		var source SourceType
		var err error

		// Use context-aware resolution if context is provided
		if ctx != nil {
			value, source, err = c.resolveValueWithPriorityContext(key, def, ctx)
		} else {
			value, source, err = c.resolveValueWithPriority(key, def)
		}

		if err != nil {
			displayValue := ""
			if value != nil && !def.secret {
				displayValue = fmt.Sprintf("%v", value)
			} else if value != nil && def.secret {
				displayValue = maskSecret(fmt.Sprintf("%v", value))
			}
			errs = append(errs, ConfigError{
				Key:              key,
				Source:           source.String(),
				Value:            displayValue,
				Display:          buildDefinitionDisplay(def),
				ErrorDescription: err.Error(),
			})
			continue
		}

		if def.secret && value != nil {
			strValue := fmt.Sprintf("%v", value)
			c.secrets.Store(key, strValue)
		} else {
			c.values[key] = value
		}
	}

	return errs
}

// processConfigWithContext parses flags from the provided args, validates all definitions,
// and populates the Config's values and secrets maps with context awareness.
func (c *Config) processConfigWithContext(args []string, ctx *CommandContext) []ConfigError {
	if c.processed {
		c.values = make(map[string]any)
		c.secrets.DestroyAll()
		c.secrets = newSecretStore()
	}
	c.processed = true

	services := c.createServices()
	flagParser := services.FlagParser
	parsedFlags, err := flagParser.ParseGlobal(args, c.definitions)
	if err != nil {
		return []ConfigError{{
			Key:              "flag_parsing",
			Source:           "flag",
			Display:          "",
			ErrorDescription: fmt.Sprintf("Flag parsing error: %v", err),
		}}
	}

	c.flagValues = parsedFlags.Values

	// Use context-aware processing if context is provided
	if ctx != nil {
		return c.processDefinitionsWithContext(ctx)
	}

	return c.processDefinitions()
}

// Destroy cleans up all secrets from memory
func (c *Config) Destroy() {
	c.secrets.DestroyAll()
}

// IsSecret checks if a configuration key is defined as a secret
func (c *Config) IsSecret(key string) bool {
	if def, exists := c.definitions[key]; exists {
		return def.secret
	}
	return false
}

// Dump returns a map of all configuration values (secrets masked)
func (c *Config) Dump() map[string]string {
	result := make(map[string]string)
	for key, def := range c.definitions {
		if def.secret {
			if c.secrets.Get(key).IsSet() {
				result[key] = "[SECRET:" + fmt.Sprintf("%d", c.secrets.Get(key).Size()) + " bytes]"
			} else {
				result[key] = "[SECRET:not set]"
			}
		} else if val, ok := c.values[key]; ok && val != nil {
			result[key] = fmt.Sprintf("%v", val)
		} else {
			result[key] = "[not set]"
		}
	}
	return result
}

// GenerateHelp creates a help message using the new template-based help system
func (c *Config) GenerateHelp() string {
	text, _ := c.getHelpService().GenerateHelp([]string{"--help"}, c.commands)
	return text
}

// getHelpService returns the help service instance, creating it if needed
func (c *Config) getHelpService() *helpService {
	if c.helpService == nil {
		c.helpService = newHelpService()
		c.helpService.SetOutput(&ConsoleHelpOutput{W: c.stdout})
		if c.name != "" {
			c.helpService.coordinator.executable = c.name
		}
	}
	return c.helpService
}

// getCommands returns the commands map for help system
func (c *Config) getCommands() map[string]*Command {
	return c.commands
}

// createServices creates a new CommandServices instance for internal use
func (c *Config) createServices() *CommandServices {
	return newCommandServices()
}

func (c *Config) Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no arguments provided")
	}

	// Check if this is a no-command application
	if len(c.commands) == 0 {
		// Create a temporary context to check for help request
		tempCtx := NewCommandContext(args[1:], c, "", "")

		errs := c.processConfigWithContext(args[1:], tempCtx)
		if len(errs) > 0 {
			// If help is requested, don't show configuration errors
			if tempCtx.IsHelpRequested() {
				// Show help instead
				return c.ShowGlobalHelp()
			}

			executable := filepath.Base(args[0])
			execCtx := NewExecutionContext(executable)
			for _, configErr := range errs {
				execCtx.CollectConfigError(c, configErr)
			}
			helpText, err := execCtx.renderErrorsWithCommand(nil, c.getHelpService())
			if err != nil {
				return err
			}
			fmt.Fprintln(c.errOut(), helpText)
			return usageError(fmt.Errorf("configuration errors"), true)
		}
		return nil
	}

	// Create services for routing
	services := c.createServices()
	router := services.CommandRouter

	// Route command with integrated help handling
	cmd, ctx, err := router.RouteWithHelpHandling(args, c)
	if err != nil {
		return err // usage errors already carry ExitUsage
	}

	// If no command to execute (help was shown), return success
	if cmd == nil {
		return nil
	}

	// Resolve global definitions (env/file/default plus any defined global
	// flags in the command args). Command-specific flags are not declared
	// globally, so they are ignored here and processed by ProcessCommandConfig.
	if ctx.execution == nil {
		ctx.execution = NewExecutionContext(ctx.Command)
	}
	for _, configErr := range c.processConfigWithContext(ctx.Args, ctx) {
		ctx.execution.CollectConfigError(c, configErr)
	}

	// Execute command with global middleware
	return c.executeWithGlobalMiddleware(cmd, ctx)
}

// executeWithGlobalMiddleware wraps command execution with global middleware
func (c *Config) executeWithGlobalMiddleware(cmd *Command, ctx *CommandContext) error {
	// Create services for middleware handling
	services := c.createServices()
	middlewareChain := services.MiddlewareChain

	// Create the final execution function that runs the command
	execFunc := func(ctx *CommandContext) error {
		result := cmd.Execute(ctx)
		if result.Error != nil {
			// Check if execution context has errors and display them
			if ctx.execution != nil && ctx.execution.HasErrors() {
				helpText, err := ctx.execution.renderErrorsWithCommand(cmd, c.getHelpService())
				if err != nil {
					return err
				}
				fmt.Fprintln(c.errOut(), helpText)
				return usageError(result.Error, true)
			}

			// Always display the message if it exists
			if result.Message != "" {
				fmt.Fprintln(c.errOut(), result.Message)
			}

			if result.ShouldExit {
				return &ExitError{Code: result.ExitCode, Err: result.Error, Reported: result.Message != ""}
			}
		}
		return result.Error
	}

	// Apply global middleware using MiddlewareChain service
	finalFunc := middlewareChain.ApplyGlobalOnly(c.globalMiddleware, execFunc)

	return finalFunc(ctx)
}

// ShowGlobalHelp displays help for all commands using the new template-based help system
func (c *Config) ShowGlobalHelp() error {
	return c.getHelpService().ShowHelp([]string{"--help"}, c.commands)
}

// ShowCommandHelp displays help for a specific command using the new template-based help system
func (c *Config) ShowCommandHelp(commandName string) error {
	return c.getHelpService().ShowHelp([]string{"app", commandName, "--help"}, c.commands)
}

// findSuggestions finds similar command names for suggestions
func (c *Config) findSuggestions(input string) string {
	var suggestions []string
	minDistance := 3

	for _, name := range slices.Sorted(maps.Keys(c.commands)) {
		if name == "" {
			continue // the default command has no name to suggest
		}
		distance := levenshteinDistance(input, name)
		if distance <= minDistance {
			suggestions = append(suggestions, name)
		}
	}

	if len(suggestions) == 0 {
		return "no similar commands found"
	}

	return strings.Join(suggestions, ", ")
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	for i := range len(a) + 1 {
		matrix[i][0] = i
	}
	for j := range len(b) + 1 {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}
