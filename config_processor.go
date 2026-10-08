// cli/config_processor.go
package kli

import (
	"fmt"
)

// ConfigProcessor processes command-specific configuration
type ConfigProcessor interface {
	// ProcessCommandConfig handles command-specific configuration processing
	ProcessCommandConfig(cmd *Command, ctx *CommandContext) *CommandResult

	// ValidateRequiredFlags checks if all required flags have values and logs warnings for missing ones
	ValidateRequiredFlags(cmd *Command, ctx *CommandContext) *CommandResult
}

// configProcessor implements ConfigProcessor interface
type configProcessor struct{}

// newConfigProcessor creates a new ConfigProcessor instance for internal use
func newConfigProcessor() ConfigProcessor {
	return &configProcessor{}
}

// ProcessCommandConfig handles command-specific configuration processing
func (cp *configProcessor) ProcessCommandConfig(cmd *Command, ctx *CommandContext) *CommandResult {
	if cmd == nil {
		return errorResult(fmt.Errorf("command cannot be nil"))
	}

	if ctx == nil {
		return errorResult(fmt.Errorf("context cannot be nil"))
	}

	// Parse command-specific flags to detect flag errors with rich reporting
	services := newCommandServices()
	flagParser := services.FlagParser
	parsedFlags, err := flagParser.ParseCommand(ctx.Args, cmd.Definitions)
	if parsedFlags != nil && parsedFlags.FlagSet != nil {
		ctx.positional = parsedFlags.FlagSet.Args()
	}

	// Create temp config with command definitions and inherited global settings
	tempConfig := &Config{
		definitions:     cmd.Definitions,
		values:          make(map[string]any),
		secrets:         newSecretStore(),
		flagValues:      parsedFlags.Values,
		fileConfig:      ctx.GlobalConfig.fileConfig,
		env:             ctx.GlobalConfig.env,
		commands:        ctx.GlobalConfig.commands,
		defaultPriority: ctx.GlobalConfig.defaultPriority,
	}

	// Handle flag parsing errors with rich per-flag error info
	if err != nil || len(parsedFlags.Errors) > 0 {
		var allErrors []error
		if err != nil {
			allErrors = append(allErrors, err)
		}
		allErrors = append(allErrors, parsedFlags.Errors...)

		flagConfigErrs := flagParser.ConvertFlagErrorsToConfigErrors(allErrors, cmd.Definitions)

		if ctx.execution != nil {
			ctx.execution.Clear()
		}
		for _, configErr := range flagConfigErrs {
			ctx.execution.CollectConfigError(tempConfig, configErr)
		}
		return configErrorResult("configuration errors detected")
	}

	if ctx.execution != nil {
		ctx.execution.Clear()
	}

	// Use the shared definition processing loop with context awareness (no duplication)
	configErrs := tempConfig.processDefinitionsWithContext(ctx)

	if len(configErrs) > 0 {
		for _, configErr := range configErrs {
			ctx.execution.CollectConfigError(tempConfig, configErr)
		}

		// If help is requested, don't return an error even if there are config errors
		if ctx.IsHelpRequested() {
			// Still set the command config so help can access definition information
			ctx.CommandConfig = tempConfig
			return success()
		}

		return errorResult(fmt.Errorf("configuration errors detected"))
	}

	ctx.CommandConfig = tempConfig
	return success()
}

// ValidateRequiredFlags checks if all required flags have values and logs warnings for missing ones
func (cp *configProcessor) ValidateRequiredFlags(cmd *Command, ctx *CommandContext) *CommandResult {
	if cmd == nil {
		return success() // Nothing to validate
	}

	if ctx == nil {
		return errorResult(fmt.Errorf("context cannot be nil"))
	}

	for key, def := range cmd.Definitions {
		if def.required {
			// Check if value is provided in any source (flag, env, or default)
			hasValue := false

			// Check flag value
			if def.flag != "" {
				var flagVal *string
				if ctx.CommandConfig != nil {
					flagVal, _ = ctx.CommandConfig.flagValues[key]
				} else {
					// Fall back to global config when no command config
					flagVal, _ = ctx.GlobalConfig.flagValues[key]
				}
				if flagVal != nil && *flagVal != "" {
					hasValue = true
				}
			}

			// Check environment variable
			if !hasValue && def.envVar != "" {
				if envVal := ctx.GlobalConfig.getenv(def.envVar); envVal != "" {
					hasValue = true
				}
			}

			// Check default value
			if !hasValue && def.defaultValue != nil {
				hasValue = true
			}

			// Log warning if required flag is missing
			if !hasValue {
				displayName := buildDefinitionDisplay(def)
				logWarningForDesigner(fmt.Sprintf("Required configuration '%s' is not provided (key: %s)", displayName, key))
			}
		}
	}

	return success()
}
