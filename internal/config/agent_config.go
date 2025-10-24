package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentYAMLConfig represents the YAML configuration for an agent.
// This is the structure that users can customize to define their agents.
type AgentYAMLConfig struct {
	// Agent identification
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	// The prompt content for this agent
	Prompt string `yaml:"prompt"`

	// Model configuration - either a type (large/small) or specific provider/model
	Model AgentModelConfig `yaml:"model"`

	// Tools configuration
	Tools AgentToolsConfig `yaml:"tools,omitempty"`

	// MCP configuration
	MCP AgentMCPConfig `yaml:"mcp,omitempty"`

	// LSP configuration
	LSP AgentLSPConfig `yaml:"lsp,omitempty"`

	// Context paths
	ContextPaths []string `yaml:"context_paths,omitempty"`

	// Whether this agent is disabled
	Disabled bool `yaml:"disabled,omitempty"`
}

// AgentModelConfig configures which model the agent should use.
type AgentModelConfig struct {
	// Type can be "large" or "small" to use the configured model type
	Type string `yaml:"type,omitempty"`

	// Specific provider and model (overrides Type)
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

// AgentToolsConfig configures which tools are available to the agent.
type AgentToolsConfig struct {
	// Allowed is a list of tool names that are allowed (whitelist mode)
	// If empty, all tools are allowed except those in Disabled
	Allowed []string `yaml:"allowed,omitempty"`

	// Disabled is a list of tool names that are disabled (blacklist mode)
	// Only used if Allowed is empty
	Disabled []string `yaml:"disabled,omitempty"`
}

// AgentMCPConfig configures which MCP servers and tools are available.
type AgentMCPConfig struct {
	// Allowed maps MCP server names to lists of allowed tool names
	// If the list is nil or empty, all tools from that server are allowed
	// If Allowed is nil (not set), all MCP servers and tools are available
	Allowed map[string][]string `yaml:"allowed,omitempty"`
}

// AgentLSPConfig configures which LSP servers are available.
type AgentLSPConfig struct {
	// Allowed is a list of LSP server names that are allowed
	// If nil or empty, all LSP servers are available
	Allowed []string `yaml:"allowed,omitempty"`
}

// LoadAgentConfig loads an agent configuration from a YAML file.
func LoadAgentConfig(path string) (*AgentYAMLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config: %w", err)
	}

	var config AgentYAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse agent config: %w", err)
	}

	return &config, nil
}

// SaveAgentConfig saves an agent configuration to a YAML file.
func SaveAgentConfig(path string, config *AgentYAMLConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal agent config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create agent config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write agent config: %w", err)
	}

	return nil
}

// ToAgent converts an AgentYAMLConfig to the internal Agent type.
func (a *AgentYAMLConfig) ToAgent() Agent {
	agent := Agent{
		ID:           a.ID,
		Name:         a.Name,
		Description:  a.Description,
		Disabled:     a.Disabled,
		ContextPaths: a.ContextPaths,
	}

	// Set model type - default to large if not specified
	if a.Model.Type != "" {
		agent.Model = SelectedModelType(a.Model.Type)
	} else {
		agent.Model = SelectedModelTypeLarge
	}

	// Set allowed tools
	if len(a.Tools.Allowed) > 0 {
		agent.AllowedTools = a.Tools.Allowed
	}

	// Set MCP configuration
	if a.MCP.Allowed != nil {
		agent.AllowedMCP = a.MCP.Allowed
	}

	// Set LSP configuration
	if a.LSP.Allowed != nil {
		agent.AllowedLSP = a.LSP.Allowed
	}

	return agent
}

// AgentsConfigDir returns the directory where agent configs are stored.
func AgentsConfigDir() string {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, appName, "agents")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "agents")
	}

	return filepath.Join(homeDir, ".config", appName, "agents")
}

// LoadAgentsFromDirectory loads all agent configurations from the agents directory.
func LoadAgentsFromDirectory() (map[string]Agent, map[string]string, error) {
	agentsDir := AgentsConfigDir()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Check if directory exists and has any yaml files
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	// If no YAML files exist, create defaults
	hasYAML := false
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
			hasYAML = true
			break
		}
	}

	if !hasYAML {
		if err := createDefaultAgentConfigs(agentsDir); err != nil {
			return nil, nil, fmt.Errorf("failed to create default agent configs: %w", err)
		}
		// Re-read directory
		entries, err = os.ReadDir(agentsDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read agents directory after creating defaults: %w", err)
		}
	}

	agents := make(map[string]Agent)
	prompts := make(map[string]string)

	// Load all YAML files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(agentsDir, entry.Name())
		config, err := LoadAgentConfig(path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load agent config %s: %w", entry.Name(), err)
		}

		if config.ID == "" {
			continue // Skip configs without ID
		}

		agents[config.ID] = config.ToAgent()
		prompts[config.ID] = config.Prompt
	}

	return agents, prompts, nil
}

// createDefaultAgentConfigs creates default agent configuration files.
func createDefaultAgentConfigs(agentsDir string) error {
	defaults := []AgentYAMLConfig{
		{
			ID:          "coder",
			Name:        "Coder",
			Description: "An agent that helps with executing coding tasks.",
			Prompt:      getDefaultCoderPrompt(),
			Model: AgentModelConfig{
				Type: "large",
			},
			Tools: AgentToolsConfig{
				Allowed: allToolNames(),
			},
			ContextPaths: defaultContextPaths,
		},
		{
			ID:          "task",
			Name:        "Task",
			Description: "An agent that helps with searching for context and finding implementation details.",
			Prompt:      getDefaultTaskPrompt(),
			Model: AgentModelConfig{
				Type: "large",
			},
			Tools: AgentToolsConfig{
				Allowed: []string{"glob", "grep", "ls", "sourcegraph", "view"},
			},
			MCP: AgentMCPConfig{
				Allowed: map[string][]string{}, // No MCPs by default
			},
			LSP: AgentLSPConfig{
				Allowed: []string{}, // No LSPs by default
			},
			ContextPaths: defaultContextPaths,
		},
	}

	for _, config := range defaults {
		path := filepath.Join(agentsDir, fmt.Sprintf("%s.yaml", config.ID))
		if err := SaveAgentConfig(path, &config); err != nil {
			return err
		}
	}

	return nil
}

// getDefaultCoderPrompt returns the default prompt for the coder agent.
func getDefaultCoderPrompt() string {
	return `You are Tulpa, an interactive CLI tool that helps users with software engineering tasks.

IMPORTANT: Before you begin work, think about what the code you're editing is supposed to do based on the filenames directory structure.

# Memory

If the current working directory contains a file called TULPA.md, it will be automatically added to your context. This file serves multiple purposes:

1. Storing frequently used bash commands (build, test, lint, etc.) so you can use them without searching each time
2. Recording the user's code style preferences (naming conventions, preferred libraries, etc.)
3. Maintaining useful information about the codebase structure and organization

When you spend time searching for commands to typecheck, lint, build, or test, you should ask the user if it's okay to add those commands to tulpa.md.

# Tone and style

You should be concise, direct, and to the point. Output text to communicate with the user; all text you output outside of tool use is displayed to the user. Only use tools to complete tasks. Never use tools like Bash or code comments as means to communicate with the user during the session.

IMPORTANT: You should minimize output tokens while maintaining helpfulness, quality, and accuracy.
IMPORTANT: You should NOT answer with unnecessary preamble or postamble.
IMPORTANT: Keep your responses short. You MUST answer concisely with fewer than 4 lines (not including tool use or code generation).`
}

// getDefaultTaskPrompt returns the default prompt for the task agent.
func getDefaultTaskPrompt() string {
	return `You are an agent for Tulpa. Given the user's prompt, you should use the tools available to you to answer the user's question.

Notes:
1. IMPORTANT: You should be concise, direct, and to the point, since your responses will be displayed on a command line interface.
2. When relevant, share file names and code snippets relevant to the query
3. Any file paths you return in your final response MUST be absolute. DO NOT use relative paths.`
}
