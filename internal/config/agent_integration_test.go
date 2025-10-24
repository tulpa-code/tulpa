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
		customAgent := `name: Custom Agent
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

		err = cfg.SetupAgents()
		require.NoError(t, err)

		require.NotNil(t, cfg.Agents)
		require.NotNil(t, cfg.AgentPrompts)

		// Verify custom agent was loaded
		require.Contains(t, cfg.Agents, "custom-agent")
		customAgentConfig := cfg.Agents["custom-agent"]
		require.Equal(t, "Custom Agent", customAgentConfig.Name)
		require.Equal(t, SelectedModelTypeSmall, customAgentConfig.Model)
		require.Equal(t, []string{"view", "grep"}, customAgentConfig.AllowedTools)

		// Verify prompt was loaded
		require.Contains(t, cfg.AgentPrompts, "custom-agent")
		require.Equal(t, "Custom prompt for testing", cfg.AgentPrompts["custom-agent"])
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
		agent := `name: Filtered Agent
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

		err = cfg.SetupAgents()
		require.NoError(t, err)

		filteredAgent := cfg.Agents["filtered-agent"]
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
		agent := `name: No Context Agent
prompt: Test
`
		err = os.WriteFile(filepath.Join(agentsDir, "no-context.yaml"), []byte(agent), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Options: &Options{
				ContextPaths: []string{".cursorrules", "TULPA.md"},
			},
		}

		err = cfg.SetupAgents()
		require.NoError(t, err)

		noContextAgent := cfg.Agents["no-context-agent"]
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
		agent := `name: Custom Context Agent
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

		err = cfg.SetupAgents()
		require.NoError(t, err)

		customContextAgent := cfg.Agents["custom-context-agent"]
		require.Equal(t, []string{"custom1.md", "custom2.md"}, customContextAgent.ContextPaths)
	})

	t.Run("returns error when YAML configs fail to load", func(t *testing.T) {
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

		// Create an invalid YAML file
		invalidYAML := `id: broken
invalid: yaml: syntax: [[[
`
		err = os.WriteFile(filepath.Join(agentsDir, "broken.yaml"), []byte(invalidYAML), 0o644)
		require.NoError(t, err)

		cfg := &Config{
			Options: &Options{
				ContextPaths:  []string{},
				DisabledTools: []string{},
			},
		}

		// Should return error, not fall back to hardcoded
		err = cfg.SetupAgents()
		require.Error(t, err)
		require.Contains(t, err.Error(), "agent configuration error")
	})
}

