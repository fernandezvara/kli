// cli/help_templates_test.go
package kli

import (
	"slices"
	"strings"
	"testing"
)

func TestNewTemplateComposer(t *testing.T) {
	composer := newTemplateComposer()

	if composer == nil {
		t.Fatal("Expected non-nil composer")
	}

	if composer.partials == nil {
		t.Error("Expected non-nil partials map")
	}

	if composer.cache == nil {
		t.Error("Expected non-nil cache map")
	}
}

func TestTemplateComposer_RegisterPartial(t *testing.T) {
	composer := newTemplateComposer()

	// Test registering a new partial
	composer.RegisterPartial("test", "test template")

	partial, err := composer.GetPartial("test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if partial != "test template" {
		t.Errorf("Expected 'test template', got '%s'", partial)
	}

	// Test that cache is cleared after registering partial
	if len(composer.cache) != 0 {
		t.Error("Expected cache to be cleared after registering partial")
	}
}

func TestTemplateComposer_GetPartial(t *testing.T) {
	composer := newTemplateComposer()

	// Test getting existing partial
	partial, err := composer.GetPartial("usage")
	if err != nil {
		t.Errorf("Unexpected error getting usage partial: %v", err)
	}

	if !strings.Contains(partial, "Usage:") {
		t.Error("Expected usage partial to contain 'Usage:'")
	}

	// Test getting non-existent partial
	_, err = composer.GetPartial("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent partial")
	}

	expectedError := "partial template 'nonexistent' not found"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestTemplateComposer_ComposeTemplate(t *testing.T) {
	composer := newTemplateComposer()

	// Test basic template composition (no errors, essential mode)
	template := composer.ComposeTemplate(false, false)

	if !strings.Contains(template, "Usage:") {
		t.Error("Expected template to contain usage section")
	}

	if !strings.Contains(template, "Flags:") {
		t.Error("Expected template to contain flags section")
	}

	if !strings.Contains(template, "Environment Variables:") {
		t.Error("Expected template to contain environment variables section")
	}

	if !strings.Contains(template, "Subcommands:") {
		t.Error("Expected template to contain subcommands section")
	}

	// Should not contain errors section
	if strings.Contains(template, "Configuration errors:") {
		t.Error("Expected template to not contain errors section when hasErrors=false")
	}
}

func TestTemplateComposer_ComposeTemplate_WithErrors(t *testing.T) {
	composer := newTemplateComposer()

	// Test template composition with errors
	template := composer.ComposeTemplate(true, false)

	if !strings.Contains(template, "Configuration errors:") {
		t.Error("Expected template to contain errors section when hasErrors=true")
	}
}

func TestTemplateComposer_ComposeTemplate_FullMode(t *testing.T) {
	composer := newTemplateComposer()

	// Test template composition in full mode
	template := composer.ComposeTemplate(false, true)

	// Should contain environment variables template (same for both modes now)
	if !strings.Contains(template, "{{if .EnvVars}}") {
		t.Error("Expected template to contain environment variables section in full mode")
	}

	// Should not contain old essential environment variables template
	if strings.Contains(template, "{{if .RequiredEnvVars}}") {
		t.Error("Expected template to not contain old essential environment variables section in full mode")
	}
}

func TestTemplateComposer_ComposeGlobalTemplate(t *testing.T) {
	composer := newTemplateComposer()

	template := composer.ComposeGlobalTemplate()

	if !strings.Contains(template, "Usage:") {
		t.Error("Expected global template to contain usage section")
	}

	if !strings.Contains(template, "Available commands:") {
		t.Error("Expected global template to contain available commands section")
	}
}

func TestTemplateComposer_ComposeErrorTemplate(t *testing.T) {
	composer := newTemplateComposer()

	template := composer.ComposeErrorTemplate()

	if !strings.Contains(template, "Usage:") {
		t.Error("Expected error template to contain usage section")
	}

	if !strings.Contains(template, "Configuration errors:") {
		t.Error("Expected error template to contain errors section")
	}

	if !strings.Contains(template, "Flags:") {
		t.Error("Expected error template to contain flags section")
	}

	if !strings.Contains(template, "Environment Variables:") {
		t.Error("Expected error template to contain environment variables section")
	}
}

func TestTemplateComposer_ListPartials(t *testing.T) {
	composer := newTemplateComposer()

	partials := composer.ListPartials()

	if len(partials) == 0 {
		t.Error("Expected at least one partial")
	}

	// Check that expected partials are present
	expectedPartials := []string{"usage", "description", "flags", "envvars", "errors", "subcommands", "global_commands"}

	for _, expected := range expectedPartials {
		found := slices.Contains(partials, expected)
		if !found {
			t.Errorf("Expected partial '%s' to be in the list", expected)
		}
	}
}

func TestTemplateComposer_ClearCache(t *testing.T) {
	composer := newTemplateComposer()

	// Generate some templates to populate cache
	composer.ComposeTemplate(false, false)
	composer.ComposeTemplate(true, true)
	composer.ComposeGlobalTemplate()

	// Verify cache has entries
	if len(composer.cache) == 0 {
		t.Error("Expected cache to have entries after template composition")
	}

	// Clear cache
	composer.ClearCache()

	// Verify cache is empty
	if len(composer.cache) != 0 {
		t.Error("Expected cache to be empty after clearing")
	}
}

func TestTemplateComposer_ValidatePartials(t *testing.T) {
	composer := newTemplateComposer()

	// Test validation with all required partials present
	err := composer.ValidatePartials()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	// Remove a required partial and test validation
	delete(composer.partials, "usage")

	err = composer.ValidatePartials()
	if err == nil {
		t.Error("Expected validation error when required partial is missing")
	}

	if !strings.Contains(err.Error(), "required partial 'usage' is missing") {
		t.Errorf("Expected error message to mention missing usage partial, got: %v", err)
	}
}

func TestTemplateComposer_Caching(t *testing.T) {
	composer := newTemplateComposer()

	// Compose template first time
	template1 := composer.ComposeTemplate(false, false)

	// Compose same template second time
	template2 := composer.ComposeTemplate(false, false)

	// Should be identical (cached)
	if template1 != template2 {
		t.Error("Expected templates to be identical when using cache")
	}

	// Verify cache has entry
	if len(composer.cache) == 0 {
		t.Error("Expected cache to have entry after template composition")
	}
}
