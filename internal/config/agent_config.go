package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type AgentYAMLConfig struct {
	Name         string           `yaml:"name"`
	Description  string           `yaml:"description"`
	Prompt       string           `yaml:"prompt"`
	Model        AgentModelConfig `yaml:"model"`
	Tools        AgentToolsConfig `yaml:"tools,omitempty"`
	MCP          AgentMCPConfig   `yaml:"mcp,omitempty"`
	LSP          AgentLSPConfig   `yaml:"lsp,omitempty"`
	Subagents    AgentSubagentsConfig `yaml:"subagents,omitempty"`
	ContextPaths []string         `yaml:"context_paths,omitempty"`
	Disabled     bool             `yaml:"disabled,omitempty"`
}

type AgentModelConfig struct {
	Type     string `yaml:"type,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

type AgentToolsConfig struct {
	Allowed  []string `yaml:"allowed,omitempty"`
	Disabled []string `yaml:"disabled,omitempty"`
}

type AgentMCPConfig struct {
	Allowed map[string][]string `yaml:"allowed,omitempty"`
}

type AgentLSPConfig struct {
	Allowed []string `yaml:"allowed,omitempty"`
}

type AgentSubagentsConfig struct {
	Allowed []string `yaml:"allowed,omitempty"`
	Default string   `yaml:"default,omitempty"`
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

func (a *AgentYAMLConfig) GenerateID() string {
	return strings.ToLower(strings.ReplaceAll(a.Name, " ", "-"))
}

func (a *AgentYAMLConfig) ToAgent() Agent {
	agent := Agent{
		ID:           a.GenerateID(),
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

	// Set subagents configuration
	if a.Subagents.Allowed != nil {
		agent.AllowedSubagents = a.Subagents.Allowed
	}
	agent.DefaultSubagent = a.Subagents.Default

	return agent
}

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

func LoadAgentsFromDirectory() (map[string]Agent, map[string]string, error) {
	agentsDir := AgentsConfigDir()

	// Create directory if it doesn't exist

	// Create directory if it doesn't exist
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create agents directory %s: %w", agentsDir, err)
	}

	// Check if directory exists and has any yaml files
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read agents directory %s: %w", agentsDir, err)
	}

	// Count YAML files

	// Count YAML files
	yamlFiles := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
			yamlFiles = append(yamlFiles, entry.Name())
		}
	}

	// If no YAML files exist, create defaults (unless in test mode)
	if len(yamlFiles) == 0 && os.Getenv("TULPA_SKIP_DEFAULT_AGENTS") == "" {
		if err := createDefaultAgentConfigs(agentsDir); err != nil {
			return nil, nil, fmt.Errorf("failed to create default agent configs in %s: %w", agentsDir, err)
		}
		// Re-read directory
		entries, err = os.ReadDir(agentsDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read agents directory after creating defaults: %w", err)
		}
	}

	agents := make(map[string]Agent)
	prompts := make(map[string]string)
	var loadErrors []string

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
			// Collect detailed error information
			loadErrors = append(loadErrors, fmt.Sprintf("  - %s: %v", entry.Name(), err))
			continue
		}

		if config.Name == "" {
			fmt.Printf("DEBUG: Config name is empty for %s\n", entry.Name())
			loadErrors = append(loadErrors, fmt.Sprintf("  - %s: missing required field 'name'", entry.Name()))
			continue
		}

		agentID := config.GenerateID()
		agents[agentID] = config.ToAgent()
		prompts[agentID] = config.Prompt
	}

	// If we found YAML files but couldn't load any, return detailed error
	if len(loadErrors) > 0 && len(agents) == 0 {
		return nil, nil, fmt.Errorf("failed to load agent configurations from %s:\n%s\n\nPlease fix the YAML syntax errors and restart Tulpa.",
			agentsDir,
			formatErrorList(loadErrors))
	}

	// If we loaded some but not all, return partial error
	if len(loadErrors) > 0 {
		return nil, nil, fmt.Errorf("some agent configurations failed to load from %s:\n%s\n\nPlease fix the YAML syntax errors and restart Tulpa.",
			agentsDir,
			formatErrorList(loadErrors))
	}

	return agents, prompts, nil
}

func formatErrorList(errors []string) string {
	result := "Errors found:\n"
	for _, err := range errors {
		result += err + "\n"
	}
	return result
}

func createDefaultAgentConfigs(agentsDir string) error {
	defaults := []AgentYAMLConfig{
		{
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
				Allowed: map[string][]string{},
			},
			LSP: AgentLSPConfig{
				Allowed: []string{},
			},
			ContextPaths: defaultContextPaths,
		},
	}

	for _, config := range defaults {
		agentID := config.GenerateID()
		path := filepath.Join(agentsDir, fmt.Sprintf("%s.yaml", agentID))
		if err := SaveAgentConfig(path, &config); err != nil {
			return err
		}
	}

	return nil
}

func getDefaultCoderPrompt() string {
	return `You are Tulpa, a CLI tool for software engineering tasks. Be concise, direct, and correct.

# Memory & Context

If TULPA.md exists in the working directory, it's automatically loaded. Use it for:

- Build, test, lint commands
- Code style preferences
- Codebase structure notes

When discovering useful commands or patterns, ask to save them to TULPA.md.

# Communication Style

- **Concise**: Max 4 lines of text (excluding tool use/code)
- **Direct**: No preamble, postamble, or explanations unless asked
- **Markdown**: GitHub-flavored, monospace-optimized
- **No emojis, no comments** (unless requested)

Examples:
` + "```" + `
user: what's 2+2?
assistant: 4

user: list files in src/
assistant: [runs ls] foo.ts bar.ts baz.ts
` + "```" + `

# Core Principles

## 1. Make Illegal States Unrepresentable

- Use types to prevent bugs at compile time
- Domain types over primitives (UserId not string)
- Algebraic data types for precise modeling

## 2. Functional Core, Imperative Shell

- Pure functions for business logic
- Side effects at boundaries (I/O, DB, APIs)
- Easy to test, reason about, refactor

## 3. Explicit Over Implicit

- Result<T, E> over exceptions (in core logic)
- No hidden dependencies or magic
- Make errors visible in signatures

## 4. Composition Over Complexity

- Small, focused, composable functions
- Avoid deep inheritance/nesting
- Obvious over clever

# Before You Code

1. **Understand context**: Check filenames, directory structure, imports
2. **Follow conventions**: Mimic existing style, use existing libraries
3. **Never assume libraries**: Check package.json/cargo.toml/etc first
4. **Security first**: No exposed secrets, ever

# Task Execution

1. Search/understand codebase
2. Implement solution
3. Test if possible
4. **Run lint/typecheck** (if commands available)
5. **Never commit** unless explicitly asked

# Tool Usage

- Parallel execution when safe (no dependencies between calls)
- Summarize tool output for user (they don't see full responses)
- Prefer agent tool for file search (reduces context)

# Testing

- Pure functions: Test inputs → outputs
- Integration tests for I/O boundaries
- Property-based testing when appropriate

# When to Break Rules

- Performance-critical paths (profile first)
- Inherently imperative APIs (wrap functionally)
- Explain tradeoffs when breaking principles

---

**Write code that's correct, maintainable, and clear - in that order.**`
}

func getDefaultTaskPrompt() string {
	return `You are an agent for Tulpa. Given the user's prompt, you should use the tools available to you to answer the user's question.

Notes:
1. IMPORTANT: You should be concise, direct, and to the point, since your responses will be displayed on a command line interface.
2. When relevant, share file names and code snippets relevant to the query
3. Any file paths you return in your final response MUST be absolute. DO NOT use relative paths.`
}
