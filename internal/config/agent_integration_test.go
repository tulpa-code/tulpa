package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupAgents(t *testing.T) {
	t.Parallel()

	t.Run("loads agents from YAML configs", func(t *testing.T) {
		t.Parallel()

		// Save original env and restore after test
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		t.Cleanup(func() {
			if originalXDG != "" {
				os.Setenv("XDG_CONFIG_HOME", originalXDG)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})

		tmpDir := t.TempDir()
		agentsDir := filepath.Join(tmpDir, "agents")
		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		err := os.MkdirAll(agentsDir, 0o755)
		require.NoError(t, err)

		// Create a custom agent
		customAgent := `id: custom
name: Custom Agent
description: A custom agent
prompt: Custom prompt for testing
model:
  type: small
tools:
  allowed:
    - view
    - grep
context_paths:
  - custom.md
`
		err = os.WriteFile(filepath.Join(agentsDir, "custom.yaml"), []byte(customAgent), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Options: &Options{
				ContextPaths:  []string{".cursorrules"},
				DisabledTools: []string{},
			},
		}

		cfg.SetupAgents()

		require.NotNil(t, cfg.Agents)
		require.NotNil(t, cfg.AgentPrompts)

		// Verify custom agent was loaded
		require.Contains(t, cfg.Agents, "custom")
		customAgentConfig := cfg.Agents["custom"]
		require.Equal(t, "Custom Agent", customAgentConfig.Name)
		require.Equal(t, SelectedModelTypeSmall, customAgentConfig.Model)
		require.Equal(t, []string{"view", "grep"}, customAgentConfig.AllowedTools)

		// Verify prompt was loaded
		require.Contains(t, cfg.AgentPrompts, "custom")
		require.Equal(t, "Custom prompt for testing", cfg.AgentPrompts["custom"])
	})

	t.Run("applies disabled tools filter", func(t *testing.T) {
		t.Parallel()

		// Save original env and restore after test
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		t.Cleanup(func() {
			if originalXDG != "" {
				os.Setenv("XDG_CONFIG_HOME", originalXDG)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})

		tmpDir := t.TempDir()
		agentsDir := filepath.Join(tmpDir, "agents")
		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		err := os.MkdirAll(agentsDir, 0o755)
		require.NoError(t, err)

		// Create agent with tools
		agent := `id: filtered
name: Filtered Agent
prompt: Test
tools:
  allowed:
    - bash
    - edit
    - view
`
		err = os.WriteFile(filepath.Join(agentsDir, "filtered.yaml"), []byte(agent), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Options: &Options{
				DisabledTools: []string{"bash"},
			},
		}

		cfg.SetupAgents()

		filteredAgent := cfg.Agents["filtered"]
		require.NotContains(t, filteredAgent.AllowedTools, "bash")
		require.Contains(t, filteredAgent.AllowedTools, "edit")
		require.Contains(t, filteredAgent.AllowedTools, "view")
	})

	t.Run("applies default context paths when not specified", func(t *testing.T) {
		t.Parallel()

		// Save original env and restore after test
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		t.Cleanup(func() {
			if originalXDG != "" {
				os.Setenv("XDG_CONFIG_HOME", originalXDG)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})

		tmpDir := t.TempDir()
		agentsDir := filepath.Join(tmpDir, "agents")
		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		err := os.MkdirAll(agentsDir, 0o755)
		require.NoError(t, err)

		// Create agent without context paths
		agent := `id: no-context
name: No Context Agent
prompt: Test
`
		err = os.WriteFile(filepath.Join(agentsDir, "no-context.yaml"), []byte(agent), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Options: &Options{
				ContextPaths: []string{".cursorrules", "TULPA.md"},
			},
		}

		cfg.SetupAgents()

		noContextAgent := cfg.Agents["no-context"]
		require.Equal(t, []string{".cursorrules", "TULPA.md"}, noContextAgent.ContextPaths)
	})

	t.Run("keeps custom context paths when specified", func(t *testing.T) {
		t.Parallel()

		// Save original env and restore after test
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		t.Cleanup(func() {
			if originalXDG != "" {
				os.Setenv("XDG_CONFIG_HOME", originalXDG)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})

		tmpDir := t.TempDir()
		agentsDir := filepath.Join(tmpDir, "agents")
		os.Setenv("XDG_CONFIG_HOME", tmpDir)

		err := os.MkdirAll(agentsDir, 0o755)
		require.NoError(t, err)

		// Create agent with custom context paths
		agent := `id: custom-context
name: Custom Context Agent
prompt: Test
context_paths:
  - custom1.md
  - custom2.md
`
		err = os.WriteFile(filepath.Join(agentsDir, "custom-context.yaml"), []byte(agent), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Options: &Options{
				ContextPaths: []string{".cursorrules"},
			},
		}

		cfg.SetupAgents()

		customContextAgent := cfg.Agents["custom-context"]
		require.Equal(t, []string{"custom1.md", "custom2.md"}, customContextAgent.ContextPaths)
	})

	t.Run("falls back to hardcoded agents on error", func(t *testing.T) {
		t.Parallel()

		// Save original env and restore after test
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		t.Cleanup(func() {
			if originalXDG != "" {
				os.Setenv("XDG_CONFIG_HOME", originalXDG)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})

		// Point to a directory we can't create
		os.Setenv("XDG_CONFIG_HOME", "/dev/null/impossible")

		cfg := &Config{
			Options: &Options{
				ContextPaths:  []string{},
				DisabledTools: []string{},
			},
		}

		cfg.SetupAgents()

		// Should have hardcoded agents
		require.NotNil(t, cfg.Agents)
		require.Contains(t, cfg.Agents, "coder")
		require.Contains(t, cfg.Agents, "task")
	})
}

func TestSetupHardcodedAgents(t *testing.T) {
	t.Parallel()

	t.Run("creates coder and task agents", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Options: &Options{
				ContextPaths:  []string{".cursorrules"},
				DisabledTools: []string{},
			},
		}

		cfg.setupHardcodedAgents()

		require.Len(t, cfg.Agents, 2)
		require.Contains(t, cfg.Agents, "coder")
		require.Contains(t, cfg.Agents, "task")

		coderAgent := cfg.Agents["coder"]
		require.Equal(t, "Coder", coderAgent.Name)
		require.Equal(t, SelectedModelTypeLarge, coderAgent.Model)
		require.NotEmpty(t, coderAgent.AllowedTools)

		taskAgent := cfg.Agents["task"]
		require.Equal(t, "Task", taskAgent.Name)
		require.Equal(t, SelectedModelTypeLarge, taskAgent.Model)
		require.NotEmpty(t, taskAgent.AllowedTools)
		require.Contains(t, taskAgent.AllowedTools, "grep")
		require.Contains(t, taskAgent.AllowedTools, "view")
		require.NotContains(t, taskAgent.AllowedTools, "bash")
	})

	t.Run("initializes empty agent prompts", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			Options: &Options{
				ContextPaths:  []string{},
				DisabledTools: []string{},
			},
		}

		cfg.setupHardcodedAgents()

		require.NotNil(t, cfg.AgentPrompts)
		require.Empty(t, cfg.AgentPrompts)
	})
}
