package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAgentConfig(t *testing.T) {
	t.Parallel()

	t.Run("loads valid YAML config", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test-agent.yaml")

		yamlContent := `id: test-agent
name: Test Agent
description: A test agent
prompt: |
  You are a test agent.
  You help with testing.
model:
  type: large
tools:
  allowed:
    - bash
    - view
mcp:
  allowed:
    server1:
      - tool1
      - tool2
lsp:
  allowed:
    - gopls
context_paths:
  - .cursorrules
disabled: false
`
		err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
		require.NoError(t, err)

		config, err := LoadAgentConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)

		require.Equal(t, "test-agent", config.ID)
		require.Equal(t, "Test Agent", config.Name)
		require.Equal(t, "A test agent", config.Description)
		require.Contains(t, config.Prompt, "You are a test agent")
		require.Equal(t, "large", config.Model.Type)
		require.Equal(t, []string{"bash", "view"}, config.Tools.Allowed)
		require.Equal(t, []string{"tool1", "tool2"}, config.MCP.Allowed["server1"])
		require.Equal(t, []string{"gopls"}, config.LSP.Allowed)
		require.Equal(t, []string{".cursorrules"}, config.ContextPaths)
		require.False(t, config.Disabled)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		_, err := LoadAgentConfig("/non/existent/path.yaml")
		require.Error(t, err)
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")

		err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0o644)
		require.NoError(t, err)

		_, err = LoadAgentConfig(configPath)
		require.Error(t, err)
	})
}

func TestSaveAgentConfig(t *testing.T) {
	t.Parallel()

	t.Run("saves config to YAML file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "saved-agent.yaml")

		config := &AgentYAMLConfig{
			ID:          "saved-agent",
			Name:        "Saved Agent",
			Description: "An agent that was saved",
			Prompt:      "You are a saved agent.",
			Model: AgentModelConfig{
				Type: "small",
			},
			Tools: AgentToolsConfig{
				Allowed: []string{"view", "grep"},
			},
		}

		err := SaveAgentConfig(configPath, config)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(configPath)
		require.NoError(t, err)

		// Load it back and verify
		loaded, err := LoadAgentConfig(configPath)
		require.NoError(t, err)
		require.Equal(t, config.ID, loaded.ID)
		require.Equal(t, config.Name, loaded.Name)
		require.Equal(t, config.Prompt, loaded.Prompt)
		require.Equal(t, config.Model.Type, loaded.Model.Type)
	})

	t.Run("creates directory if it doesn't exist", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nested", "dir", "agent.yaml")

		config := &AgentYAMLConfig{
			ID:     "nested-agent",
			Name:   "Nested Agent",
			Prompt: "Test",
		}

		err := SaveAgentConfig(configPath, config)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(configPath)
		require.NoError(t, err)
	})
}

func TestAgentYAMLConfigToAgent(t *testing.T) {
	t.Parallel()

	t.Run("converts YAML config to Agent", func(t *testing.T) {
		t.Parallel()

		yamlConfig := &AgentYAMLConfig{
			ID:          "converter-test",
			Name:        "Converter Test",
			Description: "Tests conversion",
			Prompt:      "Test prompt",
			Model: AgentModelConfig{
				Type: "large",
			},
			Tools: AgentToolsConfig{
				Allowed: []string{"bash", "edit"},
			},
			MCP: AgentMCPConfig{
				Allowed: map[string][]string{
					"server1": {"tool1"},
				},
			},
			LSP: AgentLSPConfig{
				Allowed: []string{"rust-analyzer"},
			},
			ContextPaths: []string{".cursorrules"},
			Disabled:     true,
		}

		agent := yamlConfig.ToAgent()

		require.Equal(t, "converter-test", agent.ID)
		require.Equal(t, "Converter Test", agent.Name)
		require.Equal(t, "Tests conversion", agent.Description)
		require.Equal(t, SelectedModelTypeLarge, agent.Model)
		require.Equal(t, []string{"bash", "edit"}, agent.AllowedTools)
		require.Equal(t, []string{"tool1"}, agent.AllowedMCP["server1"])
		require.Equal(t, []string{"rust-analyzer"}, agent.AllowedLSP)
		require.Equal(t, []string{".cursorrules"}, agent.ContextPaths)
		require.True(t, agent.Disabled)
	})

	t.Run("defaults to large model when type not specified", func(t *testing.T) {
		t.Parallel()

		yamlConfig := &AgentYAMLConfig{
			ID:     "default-model-test",
			Prompt: "Test",
		}

		agent := yamlConfig.ToAgent()
		require.Equal(t, SelectedModelTypeLarge, agent.Model)
	})

	t.Run("handles empty tools config", func(t *testing.T) {
		t.Parallel()

		yamlConfig := &AgentYAMLConfig{
			ID:     "empty-tools-test",
			Prompt: "Test",
		}

		agent := yamlConfig.ToAgent()
		require.Nil(t, agent.AllowedTools)
	})
}

func TestLoadAgentsFromDirectory(t *testing.T) {
	t.Parallel()

	t.Run("loads multiple agent configs", func(t *testing.T) {
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

		// Create two agent configs
		agent1 := `id: agent1
name: Agent One
prompt: First agent
model:
  type: large
`
		agent2 := `id: agent2
name: Agent Two
prompt: Second agent
model:
  type: small
`

		err = os.WriteFile(filepath.Join(agentsDir, "agent1.yaml"), []byte(agent1), 0o644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(agentsDir, "agent2.yml"), []byte(agent2), 0o644)
		require.NoError(t, err)

		agents, prompts, err := LoadAgentsFromDirectory()
		require.NoError(t, err)
		require.Len(t, agents, 2)
		require.Len(t, prompts, 2)

		require.Contains(t, agents, "agent1")
		require.Contains(t, agents, "agent2")
		require.Equal(t, "Agent One", agents["agent1"].Name)
		require.Equal(t, "Agent Two", agents["agent2"].Name)
		require.Equal(t, "First agent", prompts["agent1"])
		require.Equal(t, "Second agent", prompts["agent2"])
	})

	t.Run("skips non-YAML files", func(t *testing.T) {
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

		// Create a YAML file and a text file
		agent := `id: valid
name: Valid Agent
prompt: Valid
`
		err = os.WriteFile(filepath.Join(agentsDir, "valid.yaml"), []byte(agent), 0o644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(agentsDir, "readme.txt"), []byte("This is not YAML"), 0o644)
		require.NoError(t, err)

		agents, prompts, err := LoadAgentsFromDirectory()
		require.NoError(t, err)
		require.Len(t, agents, 1)
		require.Len(t, prompts, 1)
		require.Contains(t, agents, "valid")
	})

	t.Run("returns error for configs without ID", func(t *testing.T) {
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

		// Create config without ID
		agent := `name: No ID Agent
prompt: Has no ID
`
		err = os.WriteFile(filepath.Join(agentsDir, "no-id.yaml"), []byte(agent), 0o644)
		require.NoError(t, err)

		agents, prompts, err := LoadAgentsFromDirectory()
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing required field 'id'")
		require.Nil(t, agents)
		require.Nil(t, prompts)
	})

	t.Run("returns error for invalid YAML syntax", func(t *testing.T) {
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

		// Create invalid YAML
		invalidYAML := `id: test
name: Test
invalid yaml syntax: [[[
`
		err = os.WriteFile(filepath.Join(agentsDir, "invalid.yaml"), []byte(invalidYAML), 0o644)
		require.NoError(t, err)

		agents, prompts, err := LoadAgentsFromDirectory()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to load agent configurations")
		require.Contains(t, err.Error(), "invalid.yaml")
		require.Nil(t, agents)
		require.Nil(t, prompts)
	})

	t.Run("returns error when some configs fail to load", func(t *testing.T) {
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

		// Create one valid and one invalid config
		validAgent := `id: valid
name: Valid Agent
prompt: Valid prompt
`
		invalidAgent := `id: invalid
invalid: yaml: [[[
`
		err = os.WriteFile(filepath.Join(agentsDir, "valid.yaml"), []byte(validAgent), 0o644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(agentsDir, "invalid.yaml"), []byte(invalidAgent), 0o644)
		require.NoError(t, err)

		agents, prompts, err := LoadAgentsFromDirectory()
		require.Error(t, err)
		require.Contains(t, err.Error(), "some agent configurations failed to load")
		require.Contains(t, err.Error(), "invalid.yaml")
		require.Nil(t, agents)
		require.Nil(t, prompts)
	})
}

func TestCreateDefaultAgentConfigs(t *testing.T) {
	t.Parallel()

	t.Run("creates default coder and task agents", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		err := createDefaultAgentConfigs(tmpDir)
		require.NoError(t, err)

		// Verify coder.yaml exists
		coderPath := filepath.Join(tmpDir, "coder.yaml")
		_, err = os.Stat(coderPath)
		require.NoError(t, err)

		// Verify task.yaml exists
		taskPath := filepath.Join(tmpDir, "task.yaml")
		_, err = os.Stat(taskPath)
		require.NoError(t, err)

		// Load and verify coder config
		coderConfig, err := LoadAgentConfig(coderPath)
		require.NoError(t, err)
		require.Equal(t, "coder", coderConfig.ID)
		require.Equal(t, "Coder", coderConfig.Name)
		require.Contains(t, coderConfig.Prompt, "Tulpa")
		require.Equal(t, "large", coderConfig.Model.Type)

		// Load and verify task config
		taskConfig, err := LoadAgentConfig(taskPath)
		require.NoError(t, err)
		require.Equal(t, "task", taskConfig.ID)
		require.Equal(t, "Task", taskConfig.Name)
		require.Contains(t, taskConfig.Prompt, "agent for Tulpa")
		require.Equal(t, "large", taskConfig.Model.Type)
		require.Contains(t, taskConfig.Tools.Allowed, "grep")
		require.Contains(t, taskConfig.Tools.Allowed, "view")
	})
}

func TestAgentsConfigDir(t *testing.T) {
	t.Parallel()

	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
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

		expectedBase := "/custom/config"
		os.Setenv("XDG_CONFIG_HOME", expectedBase)

		dir := AgentsConfigDir()
		require.Equal(t, filepath.Join(expectedBase, "tulpa", "agents"), dir)
	})

	t.Run("uses .config in home when XDG_CONFIG_HOME not set", func(t *testing.T) {
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

		os.Unsetenv("XDG_CONFIG_HOME")

		dir := AgentsConfigDir()
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		expected := filepath.Join(homeDir, ".config", "tulpa", "agents")
		require.Equal(t, expected, dir)
	})
}

func TestGetDefaultPrompts(t *testing.T) {
	t.Parallel()

	t.Run("coder prompt is not empty", func(t *testing.T) {
		t.Parallel()

		prompt := getDefaultCoderPrompt()
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "Tulpa")
	})

	t.Run("task prompt is not empty", func(t *testing.T) {
		t.Parallel()

		prompt := getDefaultTaskPrompt()
		require.NotEmpty(t, prompt)
		require.Contains(t, prompt, "agent for Tulpa")
	})
}
