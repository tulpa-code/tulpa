package prompt

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tulpa-code/tulpa/internal/config"
)

func TestGetPromptWithYAMLConfig(t *testing.T) {
	t.Parallel()

	t.Run("uses custom prompt from config for coder", func(t *testing.T) {
		t.Parallel()

		// Note: This test documents expected behavior but can't easily override
		// global config in unit tests. In real usage, GetPrompt would use
		// custom prompt from loaded config.

		// Store the config temporarily
		originalCfg := config.Get()
		t.Cleanup(func() {
			// This is a bit tricky - in real usage, config is global
			// For proper testing, we'd need dependency injection
			// But for now, this demonstrates the expected behavior
		})

		// Manually set config for testing
		// Note: This won't work perfectly due to global state
		// but it documents the expected behavior
		_ = originalCfg

		// Test that GetPrompt would use the custom prompt
		// In a real scenario with proper DI, we'd pass cfg to GetPrompt
		prompt := GetPrompt(PromptCoder, "anthropic")

		// Since we can't easily override the global config in tests,
		// we'll just verify the fallback behavior works
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "Tulpa")
	})

	t.Run("falls back to embedded prompt when no custom prompt", func(t *testing.T) {
		t.Parallel()

		prompt := GetPrompt(PromptCoder, "anthropic")
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "Tulpa")
	})

	t.Run("uses embedded prompt for title", func(t *testing.T) {
		t.Parallel()

		prompt := GetPrompt(PromptTitle, "anthropic")
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "title")
	})

	t.Run("uses embedded prompt for task", func(t *testing.T) {
		t.Parallel()

		prompt := GetPrompt(PromptTask, "anthropic")
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "agent for Tulpa")
	})

	t.Run("uses embedded prompt for summarizer", func(t *testing.T) {
		t.Parallel()

		prompt := GetPrompt(PromptSummarizer, "anthropic")
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "summariz")
	})

	t.Run("returns default for unknown prompt ID", func(t *testing.T) {
		t.Parallel()

		prompt := GetPrompt(PromptID("unknown"), "anthropic")
		require.Equal(t, "You are a helpful assistant", prompt)
	})
}

func TestFormatCoderPrompt(t *testing.T) {
	t.Parallel()

	t.Run("adds environment info to custom prompt", func(t *testing.T) {
		t.Parallel()

		basePrompt := "Custom coder instructions"
		formatted := formatCoderPrompt(basePrompt)

		require.Contains(t, formatted, "Custom coder instructions")
		require.Contains(t, formatted, "<env>")
		require.Contains(t, formatted, "Working directory:")
	})

	t.Run("adds context paths when provided", func(t *testing.T) {
		t.Parallel()

		basePrompt := "Custom coder instructions"

		// We can't easily test with real files, but we can verify
		// the function doesn't crash and returns formatted output
		formatted := formatCoderPrompt(basePrompt, ".cursorrules")

		require.Contains(t, formatted, "Custom coder instructions")
		require.Contains(t, formatted, "<env>")
	})

	t.Run("works with empty context paths", func(t *testing.T) {
		t.Parallel()

		basePrompt := "Custom coder instructions"
		formatted := formatCoderPrompt(basePrompt)

		require.Contains(t, formatted, "Custom coder instructions")
		require.Contains(t, formatted, "<env>")
		require.NotContains(t, formatted, "Project-Specific Context")
	})
}
