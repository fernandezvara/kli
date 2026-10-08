// cli/command_builder.go
package kli

import "maps"

// CommandBuilder provides a fluent API for building commands
type CommandBuilder struct {
	cmd    *Command
	config *Config
}

// newCommandBuilder creates a new command builder
func newCommandBuilder(cfg *Config, name string) *CommandBuilder {
	return &CommandBuilder{
		cmd: &Command{
			Name:        name,
			Definitions: make(map[string]*Definition),
			SubCommands: make(map[string]*Command),
			Middleware:  make([]CommandMiddleware, 0),
		},
		config: cfg,
	}
}

// Func sets the command function
func (b *CommandBuilder) Func(fn CommandFunc) *CommandBuilder {
	b.cmd.Func = fn
	return b
}

// ShortHelp sets the short help text
func (b *CommandBuilder) ShortHelp(help string) *CommandBuilder {
	b.cmd.ShortHelp = help
	return b
}

// LongHelp sets the long help text
func (b *CommandBuilder) LongHelp(help string) *CommandBuilder {
	b.cmd.LongHelp = help
	return b
}

// Aliases sets the command aliases
func (b *CommandBuilder) Aliases(aliases ...string) *CommandBuilder {
	b.cmd.Aliases = aliases
	return b
}

// Config defines command-specific configuration
func (b *CommandBuilder) Config(fn func(*CommandConfig)) *CommandBuilder {
	cmdConfig := b.createCommandConfig()
	fn(cmdConfig)
	b.mergeCommandConfig(cmdConfig)
	return b
}

// createCommandConfig creates a command config with copied global definitions
func (b *CommandBuilder) createCommandConfig() *CommandConfig {
	cmdConfig := &CommandConfig{
		Config:      b.config,
		commandName: b.cmd.Name,
	}

	// Create a copy of the config for this command
	cmdConfig.Config = &Config{
		definitions: make(map[string]*Definition),
		values:      make(map[string]any),
		secrets:     newSecretStore(),
		flagValues:  make(map[string]*string),
		fileConfig:  b.config.fileConfig,
		commands:    b.config.commands,
		processed:   false,
	}

	// Copy global definitions
	maps.Copy(cmdConfig.Config.definitions, b.config.definitions)

	return cmdConfig
}

// mergeCommandConfig merges command-specific definitions
func (b *CommandBuilder) mergeCommandConfig(cmdConfig *CommandConfig) {
	// Store command-specific definitions (merge with global)
	maps.Copy(b.cmd.Definitions, cmdConfig.Config.definitions)
}

// SubCommand adds a subcommand
func (b *CommandBuilder) SubCommand(name string) *CommandBuilder {
	subBuilder := newCommandBuilder(b.config, name)
	subCmd := subBuilder.cmd
	b.cmd.SubCommands[name] = subCmd
	return subBuilder
}

// Middleware adds middleware to this command
func (b *CommandBuilder) Middleware(middleware CommandMiddleware) *CommandBuilder {
	b.cmd.Middleware = append(b.cmd.Middleware, middleware)
	return b
}

// CommandConfig wraps Config for command-specific configuration
type CommandConfig struct {
	*Config
	commandName string
}

// Define starts a new command-specific configuration definition
func (cc *CommandConfig) Define(key string) *DefinitionBuilder {
	builder := newDefinitionBuilder(cc.Config, key)
	cc.Config.definitions[key] = builder.def
	return builder
}
