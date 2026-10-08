// cli/help_templates.go
package kli

import (
	"fmt"
	"strings"
)

// templateComposer manages template partials and composition
type templateComposer struct {
	partials map[string]string
	cache    map[string]string
}

// newTemplateComposer creates a new template composer
func newTemplateComposer() *templateComposer {
	tc := &templateComposer{
		partials: make(map[string]string),
		cache:    make(map[string]string),
	}

	// Register default partials
	tc.registerDefaultPartials()
	return tc
}

// registerDefaultPartials registers the default template partials
func (tc *templateComposer) registerDefaultPartials() {
	// Template partials
	tc.partials["usage"] = `Usage: {{.Executable}} {{.Command}}{{with .Subcommand}} {{.}}{{end}} [options]`
	tc.partials["global_usage"] = `Usage: {{.Executable}} <command> [options]`
	tc.partials["description"] = `{{.Description}}`
	tc.partials["flags"] = `{{if .Flags}}Flags:
{{range .Flags}}  {{.DisplayLine}}
{{if .Description}}        {{.Description}}
{{end}}{{end}}{{end}}`
	tc.partials["envvars"] = `{{if .EnvVars}}Environment Variables:
{{range .EnvVars}}  {{.EnvVarDisplay}}
        {{.Description}}
{{end}}{{end}}`
	tc.partials["errors"] = `{{if .Errors}}Configuration errors:
{{range .Errors}}  {{.Display}} -> {{.ErrorDescription}}
{{end}}{{end}}`
	tc.partials["subcommands"] = `{{if .Subcommands}}Subcommands:
{{range .Subcommands}}  {{printf "%-12s" .Name}} {{.Description}}{{if .Aliases}} (aliases: {{join .Aliases ", "}}){{end}}
{{end}}{{end}}`
	tc.partials["global_commands"] = `{{if .Commands}}Available commands:

{{range .Commands}}{{if .Aliases}}  {{printf "%-12s" .Name}} (aliases: {{join .Aliases ", "}}) {{.Description}}
{{else}}  {{printf "%-12s" .Name}} {{.Description}}
{{end}}{{end}}

Use '{{.Executable}} <command> --help' for command-specific help{{else}}{{if .Description}}{{.Description}}

{{end}}Use '{{.Executable}} --help' for configuration options{{end}}`
}

// RegisterPartial adds or updates a template partial
func (tc *templateComposer) RegisterPartial(name, template string) {
	tc.partials[name] = template
	// Clear cache when partials change
	tc.cache = make(map[string]string)
}

// ComposeTemplate builds a complete template from partials based on context
func (tc *templateComposer) ComposeTemplate(hasErrors, isFull bool) string {
	cacheKey := fmt.Sprintf("cmd_%t_%t", hasErrors, isFull)

	if cached, exists := tc.cache[cacheKey]; exists {
		return cached
	}

	var builder strings.Builder

	// Always include usage and description
	builder.WriteString(tc.partials["usage"])
	builder.WriteString("\n\n")
	builder.WriteString(tc.partials["description"])
	builder.WriteString("\n\n")

	// Include errors if present
	if hasErrors {
		builder.WriteString(tc.partials["errors"])
		builder.WriteString("\n\n")
	}

	// Always include flags
	builder.WriteString(tc.partials["flags"])
	builder.WriteString("\n\n")

	// Include environment variables (mode filtering happens in data extraction)
	builder.WriteString(tc.partials["envvars"])
	builder.WriteString("\n\n")

	// Include subcommands if any
	builder.WriteString(tc.partials["subcommands"])

	template := builder.String()
	tc.cache[cacheKey] = template
	return template
}

// ComposeGlobalTemplate builds the global help template
func (tc *templateComposer) ComposeGlobalTemplate() string {
	if cached, exists := tc.cache["global"]; exists {
		return cached
	}

	template := tc.partials["global_usage"] + "\n\n" + tc.partials["global_commands"]
	tc.cache["global"] = template
	return template
}

// ComposeErrorTemplate builds a template for error-only display
func (tc *templateComposer) ComposeErrorTemplate() string {
	if cached, exists := tc.cache["error"]; exists {
		return cached
	}

	var builder strings.Builder

	// Usage and description
	builder.WriteString(tc.partials["usage"])
	builder.WriteString("\n\n")
	builder.WriteString(tc.partials["description"])
	builder.WriteString("\n\n")

	// Errors section
	builder.WriteString(tc.partials["errors"])
	builder.WriteString("\n\n")

	// Flags section
	builder.WriteString(tc.partials["flags"])
	builder.WriteString("\n\n")

	// Environment variables
	builder.WriteString(tc.partials["envvars"])
	builder.WriteString("\n\n")

	// Subcommands
	builder.WriteString(tc.partials["subcommands"])

	template := builder.String()
	tc.cache["error"] = template
	return template
}

// GetPartial returns a specific template partial
func (tc *templateComposer) GetPartial(name string) (string, error) {
	if partial, exists := tc.partials[name]; exists {
		return partial, nil
	}
	return "", fmt.Errorf("partial template '%s' not found", name)
}

// ListPartials returns a list of all registered partial names
func (tc *templateComposer) ListPartials() []string {
	names := make([]string, 0, len(tc.partials))
	for name := range tc.partials {
		names = append(names, name)
	}
	return names
}

// ClearCache clears the template composition cache
func (tc *templateComposer) ClearCache() {
	tc.cache = make(map[string]string)
}

// ValidatePartials checks if all required partials are present
func (tc *templateComposer) ValidatePartials() error {
	required := []string{"usage", "global_usage", "description", "flags", "envvars", "errors", "subcommands", "global_commands"}

	for _, name := range required {
		if _, exists := tc.partials[name]; !exists {
			return fmt.Errorf("required partial '%s' is missing", name)
		}
	}
	return nil
}
