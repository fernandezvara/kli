// cli/help_data.go
package kli

import (
	"cmp"
	"maps"
	"slices"
	"strings"
)

// unifiedExtractor extracts and processes help data
type unifiedExtractor struct{}

// newUnifiedExtractor creates a new unified extractor
func newUnifiedExtractor() *unifiedExtractor {
	return &unifiedExtractor{}
}

// ExtractFlags extracts flag information from definitions
func (ue *unifiedExtractor) ExtractFlags(defs map[string]*Definition) []flagInfo {
	var flags []flagInfo

	// Sort definitions for consistent display
	for _, key := range slices.Sorted(maps.Keys(defs)) {
		def := defs[key]

		// Only include actual flags (those with a flag name), not environment-only variables
		if def.flag == "" {
			continue // Skip environment-only variables
		}

		flag := flagInfo{
			Description: def.description,
			Required:    def.required,
			EnvVar:      def.envVar,

			// Set DisplayLine for flags
			DisplayLine: buildDefinitionDisplay(def)}

		// Also set EnvVarDisplay if it has an environment variable
		if def.envVar != "" {
			flag.EnvVarDisplay = buildDefinitionDisplay(def)
		}

		flags = append(flags, flag)
	}

	return flags
}

// ExtractEnvVars extracts environment variable information from definitions
func (ue *unifiedExtractor) ExtractEnvVars(defs map[string]*Definition) []flagInfo {
	var envVars []flagInfo

	// Sort definitions for consistent display
	for _, key := range slices.Sorted(maps.Keys(defs)) {
		def := defs[key]

		// Include environment variables (those with env var name)
		if def.envVar == "" {
			continue // Skip non-environment variables
		}

		envVar := flagInfo{
			Description: def.description,
			Required:    def.required,
			EnvVar:      def.envVar,
		}

		// Set DisplayLine for environment variables
		// For environment variable display, create a modified definition without flag
		envDef := *def   // Copy the definition
		envDef.flag = "" // Clear flag to force environment variable display format
		envVar.DisplayLine = buildDefinitionDisplay(&envDef)
		envVar.EnvVarDisplay = envVar.DisplayLine

		envVars = append(envVars, envVar)
	}

	return envVars
}

// FilterEnvVars filters environment variables based on help mode
func (ue *unifiedExtractor) FilterEnvVars(envVars []flagInfo, mode helpMode) []flagInfo {
	var result []flagInfo

	for _, envVar := range envVars {
		// Only include items that actually have environment variables
		if envVar.EnvVar == "" {
			continue
		}

		if mode == helpModeFull {
			// Include all environment variables in full mode
			result = append(result, envVar)
		} else {
			// Only include required environment variables in essential mode
			if envVar.Required {
				result = append(result, envVar)
			}
		}
	}
	return result
}

// helpSummary is the one-line description of a command in a list: its
// ShortHelp, else the first line of its LongHelp.
func helpSummary(cmd *Command) string {
	if cmd.ShortHelp != "" {
		return cmd.ShortHelp
	}
	first, _, _ := strings.Cut(cmd.LongHelp, "\n")
	return first
}

// ExtractSubcommands extracts subcommand information
func (ue *unifiedExtractor) ExtractSubcommands(cmd *Command) []subcommandInfo {
	if cmd == nil || len(cmd.SubCommands) == 0 {
		return []subcommandInfo{}
	}

	var subcommands []subcommandInfo

	// Sort subcommands for consistent display
	for _, name := range slices.Sorted(maps.Keys(cmd.SubCommands)) {
		subCmd := cmd.SubCommands[name]
		desc := helpSummary(subCmd)
		subcommands = append(subcommands, subcommandInfo{
			Name:        name,
			Description: desc,
			Aliases:     subCmd.Aliases,
		})
	}

	return subcommands
}

// --- Data extraction methods for layer coordinator ---

// extractUsageData extracts usage layer data
func (ue *unifiedExtractor) extractUsageData(command, subcommand, executable string) *usageData {
	return &usageData{
		command:    command,
		subcommand: subcommand,
		executable: executable,
	}
}

// extractCommandsData extracts commands layer data
func (ue *unifiedExtractor) extractCommandsData(commands map[string]*Command, executable string) *commandsData {
	var commandSummaries []commandSummary
	for name, cmd := range commands {
		if name != "" { // Skip empty string command
			description := helpSummary(cmd)

			commandSummaries = append(commandSummaries, commandSummary{
				Name:        name,
				Description: description,
				Aliases:     cmd.Aliases,
			})
		}
	}

	slices.SortFunc(commandSummaries, func(a, b commandSummary) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &commandsData{
		commands:   commandSummaries,
		executable: executable,
	}
}

// extractFlagsData extracts flags layer data
func (ue *unifiedExtractor) extractFlagsData(cmd *Command) *flagsData {
	if cmd == nil {
		return &flagsData{}
	}

	flags := ue.ExtractFlags(cmd.Definitions)
	return &flagsData{
		flags: flags,
	}
}

// extractEnvVarsData extracts environment variables layer data
func (ue *unifiedExtractor) extractEnvVarsData(cmd *Command, mode helpMode) *envVarsData {
	if cmd == nil {
		return &envVarsData{}
	}

	envVars := ue.ExtractEnvVars(cmd.Definitions)
	filteredEnvVars := ue.FilterEnvVars(envVars, mode)

	return &envVarsData{
		envVars: filteredEnvVars,
	}
}

// extractSubcommandsData extracts subcommands layer data
func (ue *unifiedExtractor) extractSubcommandsData(cmd *Command) *subcommandsData {
	if cmd == nil {
		return &subcommandsData{}
	}

	subcommands := ue.ExtractSubcommands(cmd)
	return &subcommandsData{
		subcommands: subcommands,
	}
}

// extractErrorsData extracts errors layer data
func (ue *unifiedExtractor) extractErrorsData(errors []GetError) *errorsData {
	return &errorsData{
		errors: errors,
	}
}
