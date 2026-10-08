// cli/services.go
package kli

// CommandServices holds all command execution services
type CommandServices struct {
	Executor        CommandExecutor
	ConfigProcessor ConfigProcessor
	MiddlewareChain MiddlewareChain
	CommandRouter   CommandRouter
	FlagParser      FlagParser
}

// newCommandServices creates a new CommandServices instance with all services initialized
func newCommandServices() *CommandServices {
	return &CommandServices{
		Executor:        newCommandExecutor(),
		ConfigProcessor: newConfigProcessor(),
		MiddlewareChain: newMiddlewareChain(),
		CommandRouter:   newCommandRouter(),
		FlagParser:      newFlagParser(),
	}
}
