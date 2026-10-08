// cli/command_result.go
package kli

import "fmt"

// CommandResult represents the result of command execution with unified error handling
type CommandResult struct {
	Error      error
	ExitCode   int
	ShouldExit bool
	Message    string
}

// success creates a successful command result
func success() *CommandResult {
	return &CommandResult{
		ExitCode:   0,
		ShouldExit: false,
	}
}

// errorResult creates an error command result
func errorResult(err error) *CommandResult {
	return &CommandResult{
		Error:      err,
		ExitCode:   1,
		ShouldExit: false,
	}
}

// validationError creates a validation error result that should exit
func validationError(message string) *CommandResult {
	return &CommandResult{
		Error:      fmt.Errorf("validation error: %s", message),
		ExitCode:   ExitUsage,
		ShouldExit: true,
		Message:    message,
	}
}

// configErrorResult creates a configuration error result that should exit
func configErrorResult(message string) *CommandResult {
	return &CommandResult{
		Error:      fmt.Errorf("configuration error: %s", message),
		ExitCode:   ExitUsage,
		ShouldExit: true,
		Message:    message,
	}
}
