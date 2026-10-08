// cli/help_layer_coordinator.go
package kli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// helpCoordinator provides template-based rendering for each help layer and unified help functionality
type helpCoordinator struct {
	templates  *templateComposer
	output     helpOutput
	executable string
	extractor  *unifiedExtractor
}

// newHelpCoordinator creates a new help coordinator
func newHelpCoordinator() *helpCoordinator {
	templates := newTemplateComposer()
	output := &ConsoleHelpOutput{}

	return &helpCoordinator{
		templates:  templates,
		output:     output,
		executable: getExecutableName(),
		extractor:  newUnifiedExtractor(),
	}
}

// RenderUsage renders the usage layer using templates
func (hc *helpCoordinator) RenderUsage(data *usageData) string {
	templateStr := hc.templates.partials["usage"]

	templateData := struct {
		Command    string
		Subcommand string
		Executable string
	}{
		Command:    data.command,
		Subcommand: data.subcommand,
		Executable: data.executable,
	}

	return hc.executeTemplate(templateStr, templateData)
}

// RenderCommands renders the commands layer using templates
func (hc *helpCoordinator) RenderCommands(data *commandsData) string {
	templateStr := hc.templates.partials["global_commands"]

	templateData := struct {
		Executable string
		Commands   []commandSummary
	}{
		Executable: data.executable,
		Commands:   data.commands,
	}

	return hc.executeTemplate(templateStr, templateData)
}

// RenderFlags renders the flags layer using templates
func (hc *helpCoordinator) RenderFlags(data *flagsData) string {
	templateStr := hc.templates.partials["flags"]

	templateData := struct {
		Flags []flagInfo
	}{
		Flags: data.flags,
	}

	return hc.executeTemplate(templateStr, templateData)
}

// RenderEnvVars renders the environment variables layer using templates
func (hc *helpCoordinator) RenderEnvVars(data *envVarsData) string {
	templateStr := hc.templates.partials["envvars"]

	templateData := struct {
		EnvVars []flagInfo
	}{
		EnvVars: data.envVars,
	}

	return hc.executeTemplate(templateStr, templateData)
}

// RenderSubcommands renders the subcommands layer using templates
func (hc *helpCoordinator) RenderSubcommands(data *subcommandsData) string {
	templateStr := hc.templates.partials["subcommands"]

	templateData := struct {
		Subcommands []subcommandInfo
	}{
		Subcommands: data.subcommands,
	}

	return hc.executeTemplate(templateStr, templateData)
}

// RenderErrors renders the errors layer using templates
func (hc *helpCoordinator) RenderErrors(data *errorsData) string {
	templateStr := hc.templates.partials["errors"]

	templateData := struct {
		Errors []GetError
	}{
		Errors: data.errors,
	}

	return hc.executeTemplate(templateStr, templateData)
}

// SetOutput sets the help output destination
func (hc *helpCoordinator) SetOutput(output helpOutput) {
	hc.output = output
}

// ShowHelpWithCommands displays help with access to commands map
func (hc *helpCoordinator) ShowHelpWithCommands(command, subcommand string, full bool, errors []GetError, commands map[string]*Command) error {
	// Get the command
	var cmd *Command

	if command == "" {
		// Check if there's an empty string command first
		if commands != nil {
			if emptyCmd, exists := commands[""]; exists && len(commands) == 1 {
				// A config-only app: help is the default command's help.
				cmd = emptyCmd
			} else {
				// Otherwise list the commands (the default command's own
				// help is `app <its name> --help` or the flags of its alias).
				// No empty string command, show global help
				return hc.showGlobalHelp(commands)
			}
		} else {
			// No commands map available, show global help without commands
			return hc.showGlobalHelp(nil)
		}
	}

	if commands != nil {
		// Use provided commands map
		if subcommand != "" {
			// Subcommand help
			parentCmd, exists := commands[command]
			if !exists {
				return fmt.Errorf("parent command %s not found", command)
			}

			subCmd, exists := parentCmd.SubCommands[subcommand]
			if !exists {
				return fmt.Errorf("subcommand %s not found in command %s", subcommand, command)
			}
			cmd = subCmd
		} else {
			// Command help
			var exists bool
			cmd, exists = commands[command]
			if !exists {
				// Fall back to simple command help for unknown commands
				return hc.showSimpleCommandHelp(command, subcommand, errors)
			}
		}
	} else {
		// Fallback to simple command help without full details
		return hc.showSimpleCommandHelp(command, subcommand, errors)
	}

	// Determine help mode
	mode := helpModeEssential
	if full {
		mode = helpModeFull
	}

	// Render help using layer functions
	return hc.renderCommandHelp(cmd, command, subcommand, mode, errors)
}

// renderCommandHelp renders command help using layer functions
func (hc *helpCoordinator) renderCommandHelp(cmd *Command, command, subcommand string, mode helpMode, errors []GetError) error {
	var output strings.Builder

	// Usage layer
	usageData := hc.extractor.extractUsageData(command, subcommand, hc.executable)
	output.WriteString(hc.RenderUsage(usageData))
	output.WriteString("\n\n")

	// Description layer (only if command has description)
	if cmd.LongHelp != "" || cmd.ShortHelp != "" {
		description := cmd.ShortHelp
		if cmd.LongHelp != "" && !strings.HasPrefix(cmd.LongHelp, description) {
			if description != "" {
				description += "\n\n"
			}
			description += cmd.LongHelp
		} else if cmd.LongHelp != "" {
			description = cmd.LongHelp
		}
		output.WriteString(description)
		output.WriteString("\n\n")
	}

	// Errors layer (if any)
	if len(errors) > 0 {
		errorsData := hc.extractor.extractErrorsData(errors)
		output.WriteString(hc.RenderErrors(errorsData))
		output.WriteString("\n\n")
	}

	// Flags layer
	flagsData := hc.extractor.extractFlagsData(cmd)
	if len(flagsData.flags) > 0 {
		output.WriteString(hc.RenderFlags(flagsData))
		output.WriteString("\n\n")
	}

	// Environment variables layer
	envVarsData := hc.extractor.extractEnvVarsData(cmd, mode)
	if len(envVarsData.envVars) > 0 {
		output.WriteString(hc.RenderEnvVars(envVarsData))
		output.WriteString("\n\n")
	}

	// Subcommands layer
	subcommandsData := hc.extractor.extractSubcommandsData(cmd)
	if len(subcommandsData.subcommands) > 0 {
		output.WriteString(hc.RenderSubcommands(subcommandsData))
	}

	return hc.output.Print(output.String())
}

// showGlobalHelp displays help for all commands using template system
func (hc *helpCoordinator) showGlobalHelp(commands map[string]*Command) error {
	// Extract commands data
	commandsData := hc.extractor.extractCommandsData(commands, hc.executable)

	// Use the template system
	templateStr := hc.templates.ComposeGlobalTemplate()

	// Create template data
	templateData := struct {
		Executable  string
		Commands    []commandSummary
		Description string
	}{
		Executable:  hc.executable,
		Commands:    commandsData.commands,
		Description: "", // No description for global help
	}

	// Create template with custom functions
	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	// Execute template
	tmpl, err := template.New("global").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse global template: %w", err)
	}

	var builder strings.Builder
	err = tmpl.Execute(&builder, templateData)
	if err != nil {
		return fmt.Errorf("failed to execute global template: %w", err)
	}

	return hc.output.Print(builder.String())
}

// showSimpleCommandHelp shows basic help when commands map is not available
func (hc *helpCoordinator) showSimpleCommandHelp(command, subcommand string, errors []GetError) error {
	var output strings.Builder

	// Usage layer
	usageData := hc.extractor.extractUsageData(command, subcommand, hc.executable)
	output.WriteString(hc.RenderUsage(usageData))
	output.WriteString("\n\n")

	// Errors layer (if any)
	if len(errors) > 0 {
		errorsData := hc.extractor.extractErrorsData(errors)
		output.WriteString(hc.RenderErrors(errorsData))
		output.WriteString("\n\n")
	}

	// Help hint
	output.WriteString(fmt.Sprintf("Use %s %s", hc.executable, command))
	if subcommand != "" {
		output.WriteString(fmt.Sprintf(" %s", subcommand))
	}
	output.WriteString(" --help' or '--full-help' for more information\n")

	return hc.output.Print(output.String())
}

// executeTemplate is a helper function to execute templates with common functions
func (hc *helpCoordinator) executeTemplate(templateStr string, data any) string {
	if templateStr == "" {
		return ""
	}

	// Create template with custom functions
	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	// Parse and execute template
	tmpl, err := template.New("layer").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return fmt.Sprintf("Template error: %v", err)
	}

	var builder strings.Builder
	err = tmpl.Execute(&builder, data)
	if err != nil {
		return fmt.Sprintf("Execution error: %v", err)
	}

	return builder.String()
}

// getExecutableName returns the executable name
func getExecutableName() string {
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "cli"
}
