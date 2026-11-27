// cli/help_models.go
package kli

import (
	"slices"
	"strings"
)

// helpMode represents the help mode (currently essential vs full)
type helpMode int

const (
	helpModeEssential helpMode = iota
	helpModeFull
)

// commandSummary represents a command summary for global help
type commandSummary struct {
	Name        string
	Description string
	Aliases     []string
}

// flagInfo represents flag information for help display
type flagInfo struct {
	Description   string
	Required      bool
	EnvVar        string
	DisplayLine   string
	EnvVarDisplay string
}

// subcommandInfo represents subcommand information for help display
type subcommandInfo struct {
	Name        string
	Description string
	Aliases     []string
}

// --- Centralized help flag detection functions ---

// isHelpFlag returns true if the given argument is any help flag. Like other
// flags, help flags take one or two dashes.
func isHelpFlag(arg string) bool {
	switch arg {
	case "help", "-h", "--h", "-help", "--help":
		return true
	}
	return isFullHelpFlag(arg)
}

// isFullHelpFlag returns true if the given argument requests full/extended help
func isFullHelpFlag(arg string) bool {
	return arg == "-full-help" || arg == "--full-help"
}

// argsContainHelpFlag returns true if any argument in the slice is a help flag
func argsContainHelpFlag(args []string) bool {
	return slices.ContainsFunc(args, isHelpFlag)
}

// argsContainFullHelp returns true if any argument requests full help
func argsContainFullHelp(args []string) bool {
	return slices.ContainsFunc(args, isFullHelpFlag)
}

// lastArgIsHelpFlag returns true if the last argument is a help flag
func lastArgIsHelpFlag(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return isHelpFlag(args[len(args)-1])
}

// takesValue returns true if arg is a flag that consumes the next argument as
// its value, as in "--env prod"
func takesValue(arg string, defs map[string]*Definition) bool {
	name, dashed := strings.CutPrefix(arg, "-")
	if !dashed {
		return false
	}
	name = strings.TrimPrefix(name, "-")
	for _, def := range defs {
		if def.valueType != TypeBool && slices.Contains(def.flags, name) {
			return true
		}
	}
	return false
}

// helpRequest reports whether args ask for help, and whether for full help. A
// help flag counts in any position before "--", unless it is the value of the
// flag in front of it. As the last argument it always counts, and so does the
// word "help".
func helpRequest(args []string, defs map[string]*Definition) (requested, full bool) {
	if lastArgIsHelpFlag(args) {
		return true, argsContainFullHelp(args)
	}
	for i := 0; i < len(args) && args[i] != "--"; i++ {
		switch {
		case isHelpFlag(args[i]) && strings.HasPrefix(args[i], "-"):
			requested = true
			full = full || isFullHelpFlag(args[i])
		case takesValue(args[i], defs):
			i++
		}
	}
	return requested, full
}

// helpTopics returns the arguments that can name what help is asked about:
// those that are not flags, flag values or help flags
func helpTopics(args []string, defs map[string]*Definition) []string {
	var topics []string
	for i := 0; i < len(args) && args[i] != "--"; i++ {
		switch {
		case takesValue(args[i], defs):
			i++
		case !strings.HasPrefix(args[i], "-") && !isHelpFlag(args[i]):
			topics = append(topics, args[i])
		}
	}
	return topics
}

// usageData represents data for usage layer rendering
type usageData struct {
	command    string
	subcommand string
	executable string
}

// commandsData represents data for commands layer rendering
type commandsData struct {
	commands   []commandSummary
	executable string
}

// flagsData represents data for flags layer rendering
type flagsData struct {
	flags []flagInfo
}

// envVarsData represents data for environment variables layer rendering
type envVarsData struct {
	envVars []flagInfo
}

// subcommandsData represents data for subcommands layer rendering
type subcommandsData struct {
	subcommands []subcommandInfo
}

// errorsData represents data for errors layer rendering
type errorsData struct {
	errors []GetError
}
