// cli/help_service.go
package kli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// helpOutput handles help output destinations
type helpOutput interface {
	Print(text string) error
	Get() string
	Reset()
}

// ConsoleHelpOutput implements HelpOutput for console output
type ConsoleHelpOutput struct {
	// W is where help is written; nil means os.Stdout.
	W io.Writer
}

// Print prints text to console
func (cho *ConsoleHelpOutput) Print(text string) error {
	w := cho.W
	if w == nil {
		w = os.Stdout
	}
	_, err := io.WriteString(w, text)
	return err
}

// Get returns the accumulated output (not applicable for console)
func (cho *ConsoleHelpOutput) Get() string {
	return ""
}

// Reset resets the output (not applicable for console)
func (cho *ConsoleHelpOutput) Reset() {
	// No-op for console output
}

// StringHelpOutput captures help output as a string
type StringHelpOutput struct {
	buffer strings.Builder
}

// Print appends text to the string builder
func (sho *StringHelpOutput) Print(text string) error {
	sho.buffer.WriteString(text)
	return nil
}

// Get returns the accumulated output
func (sho *StringHelpOutput) Get() string {
	return sho.buffer.String()
}

// Reset clears the accumulated output
func (sho *StringHelpOutput) Reset() {
	sho.buffer.Reset()
}

// helpService provides help display services
type helpService struct {
	coordinator *helpCoordinator
	output      helpOutput
}

// newHelpService creates a new help service
func newHelpService() *helpService {
	return &helpService{
		coordinator: newHelpCoordinator(),
		output:      &ConsoleHelpOutput{},
	}
}

// SetOutput sets the help output destination
func (hs *helpService) SetOutput(output helpOutput) {
	hs.output = output
	hs.coordinator.SetOutput(output)
}

// GetOutput returns the current help output
func (hs *helpService) GetOutput() helpOutput {
	return hs.output
}

// ShowHelpUnified is the primary unified help display method
func (hs *helpService) ShowHelpUnified(command, subcommand string, full bool, errors []GetError, commands map[string]*Command) error {
	return hs.coordinator.ShowHelpWithCommands(command, subcommand, full, errors, commands)
}

// ShowHelp detects help type and displays appropriate help
func (hs *helpService) ShowHelp(args []string, commands map[string]*Command) error {
	// Simple help detection
	if len(args) == 0 {
		return hs.ShowGlobalHelp(commands)
	}

	if !lastArgIsHelpFlag(args) {
		return fmt.Errorf("no help requested")
	}

	isFull := argsContainFullHelp(args)

	// Find the command name by matching non-help-flag args against known commands
	if commands != nil {
		for _, arg := range args {
			if isHelpFlag(arg) {
				continue
			}
			if _, exists := commands[arg]; exists {
				return hs.ShowHelpUnified(arg, "", isFull, []GetError{}, commands)
			}
		}
	}

	return hs.ShowGlobalHelp(commands)
}

// GenerateHelp generates help text without displaying it
func (hs *helpService) GenerateHelp(args []string, commands map[string]*Command) (string, error) {
	// Capture output to string
	stringOutput := &StringHelpOutput{}
	originalOutput := hs.output
	hs.SetOutput(stringOutput)
	defer hs.SetOutput(originalOutput)

	// Use ShowHelp to generate the text
	err := hs.ShowHelp(args, commands)
	if err != nil {
		return "", err
	}

	return stringOutput.Get(), nil
}

// ShowGlobalHelp displays global help for all commands
func (hs *helpService) ShowGlobalHelp(commands map[string]*Command) error {
	return hs.coordinator.ShowHelpWithCommands("", "", false, []GetError{}, commands)
}

// ShowCommandHelp displays help for a specific command
func (hs *helpService) ShowCommandHelp(commandName string, commands map[string]*Command) error {
	return hs.coordinator.ShowHelpWithCommands(commandName, "", false, []GetError{}, commands)
}
