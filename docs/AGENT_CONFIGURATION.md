# Agent Configuration

Tulpa supports customizable agent configurations via YAML files. This allows you to define custom agents with specific prompts, tools, and capabilities tailored to your workflow.

## Configuration Location

Agent configurations are stored in:
- **Linux/macOS**: `~/.config/tulpa/agents/`
- **With XDG_CONFIG_HOME**: `$XDG_CONFIG_HOME/tulpa/agents/`

## Default Agents

On first run, Tulpa creates two default agent configurations:

### 1. Coder Agent (`coder.yaml`)
The primary agent for coding tasks with full tool access.

### 2. Task Agent (`task.yaml`)
A read-only agent for searching and finding information, with limited tool access.

## YAML Configuration Format

```yaml
# Agent identification (required)
id: my-custom-agent
name: My Custom Agent
description: A custom agent for specific tasks

# Agent prompt (required)
# This is the system prompt that guides the agent's behavior
prompt: |
  You are a specialized agent for Tulpa.
  Your role is to help users with specific tasks.
  Be concise and helpful.

# Model configuration
model:
  # Use a configured model type: "large" or "small"
  type: large

  # OR specify a specific provider and model (not yet implemented)
  # provider: openai
  # model: gpt-4o

# Tools configuration
tools:
  # Whitelist mode: only allow specific tools
  allowed:
    - bash
    - edit
    - view
    - write
    - grep
    - glob

  # OR Blacklist mode: allow all tools except these
  # disabled:
  #   - sourcegraph

# MCP (Model Context Protocol) configuration
mcp:
  # Map of MCP server names to allowed tools
  # Empty list [] means all tools from that server
  # Not specifying a server means it's not available
  allowed:
    my-mcp-server:
      - tool1
      - tool2
    another-server: []  # All tools allowed

# LSP (Language Server Protocol) configuration
lsp:
  # List of allowed LSP servers
  allowed:
    - gopls
    - rust-analyzer

  # Empty list means no LSP servers
  # Not specifying this field means all LSP servers are available

# Context paths
# Files to include in the agent's context
context_paths:
  - .cursorrules
  - TULPA.md
  - docs/style-guide.md

# Disable this agent
disabled: false
```

## Available Built-in Tools

- `bash` - Execute shell commands
- `edit` - Edit existing files
- `multiedit` - Edit multiple files at once
- `write` - Create new files
- `view` - Read file contents
- `grep` - Search file contents
- `glob` - Find files by pattern
- `ls` - List directory contents
- `sourcegraph` - Search code using Sourcegraph
- `download` - Download files from URLs
- `fetch` - Fetch content from URLs
- `agent` - Invoke sub-agents (for coder agent only)

## Example Configurations

### Code Reviewer Agent

```yaml
id: reviewer
name: Code Reviewer
description: Reviews code for style, bugs, and best practices

prompt: |
  You are a code reviewer for Tulpa.

  Your responsibilities:
  - Review code for bugs and potential issues
  - Check for style consistency
  - Suggest improvements
  - Verify security best practices

  Be thorough but constructive in your feedback.

model:
  type: large

tools:
  allowed:
    - view
    - grep
    - glob
    - ls

# Read-only: no MCP or LSP
mcp:
  allowed: {}
lsp:
  allowed: []

context_paths:
  - .cursorrules
  - docs/style-guide.md
```

### Documentation Writer

```yaml
id: docs
name: Documentation Writer
description: Writes and updates documentation

prompt: |
  You are a documentation specialist for Tulpa.

  Your focus:
  - Write clear, concise documentation
  - Follow project documentation standards
  - Include code examples where appropriate
  - Keep documentation up-to-date with code changes

  Write for developers of all skill levels.

model:
  type: large

tools:
  allowed:
    - view
    - edit
    - write
    - grep
    - glob

context_paths:
  - docs/style-guide.md
  - TULPA.md
```

### Quick Scripter

```yaml
id: scripter
name: Quick Scripter
description: Quickly creates scripts and utilities

prompt: |
  You are a scripting specialist.
  Create quick, reliable scripts for common tasks.
  Focus on: correctness, simplicity, and error handling.

model:
  type: small  # Faster for simple scripts

tools:
  allowed:
    - bash
    - write
    - view
    - edit

context_paths:
  - scripts/
```

## Customizing Existing Agents

To customize the default coder or task agents:

1. Locate the configuration files:
   ```bash
   cd ~/.config/tulpa/agents
   ```

2. Edit the YAML file:
   ```bash
   vim coder.yaml
   ```

3. Modify the prompt, tools, or other settings

4. Restart Tulpa or start a new session

## Best Practices

### Prompt Design

- **Be specific**: Clearly define the agent's role and responsibilities
- **Set expectations**: Explain how the agent should communicate
- **Include guidelines**: Add any specific rules or conventions
- **Keep it concise**: Long prompts can dilute the message

### Tool Selection

- **Principle of least privilege**: Only grant tools that the agent needs
- **Read-only for analysis**: Use view/grep/glob for review tasks
- **Full access for coding**: Use all tools for primary development agents
- **Test thoroughly**: Verify tool combinations work as expected

### Context Paths

- **Project-specific**: Include files with project conventions
- **Style guides**: Add coding style and pattern documents
- **Common commands**: Reference build, test, lint commands
- **Keep it relevant**: Too much context can be overwhelming

## Troubleshooting

### YAML Syntax Errors

**Tulpa will NOT start** if your agent configuration files have syntax errors. This is intentional to prevent unexpected behavior.

If Tulpa fails to start with an error like:
```
failed to load agent configurations from ~/.config/tulpa/agents:
Errors found:
  - coder.yaml: yaml: line 5: mapping values are not allowed in this context
  - task.yaml: missing required field 'id'

Please fix the YAML syntax errors and restart Tulpa.
```

**What to do:**

1. **Check the error message** - it will tell you exactly which files have problems
2. **Validate your YAML** - use a YAML validator or linter
3. **Common mistakes:**
   - Missing colons after keys
   - Incorrect indentation (YAML is whitespace-sensitive)
   - Mixing tabs and spaces (use spaces only)
   - Missing `id` field (required)
   - Missing `prompt` field (required)
4. **Fix the errors** and restart Tulpa

**Example of invalid YAML:**
```yaml
id: my-agent
name My Agent  # Missing colon!
prompt: |
This is my prompt
	with a tab instead of spaces  # Tabs not allowed!
```

**Example of valid YAML:**
```yaml
id: my-agent
name: My Agent  # Correct: has colon
prompt: |
  This is my prompt
  with proper indentation  # Using spaces
```

### Agent not found

If your custom agent isn't loading:

1. Check the file is in the correct directory: `~/.config/tulpa/agents/`
2. Verify the file has a `.yaml` or `.yml` extension
3. Ensure the `id` field is set and unique
4. Validate the YAML syntax (Tulpa will error on invalid YAML)

### Tools not working

If tools aren't available to your agent:

1. Check the `tools.allowed` list includes the tool
2. Verify the tool isn't in the global `disabled_tools` config
3. Check for typos in tool names

### Prompt not being used

If your custom prompt isn't being used:

1. Verify the `prompt` field is set in the YAML
2. Check for YAML syntax errors (Tulpa will fail to start)
3. Ensure the agent ID matches the one you're using

### No YAML files = Default agents

If there are NO YAML files in `~/.config/tulpa/agents/`, Tulpa will automatically create default configurations for the `coder` and `task` agents. This only happens on first run or if the directory is empty.

## Advanced: Multiple Agent Configs

You can create multiple specialized agents for different tasks:

```bash
~/.config/tulpa/agents/
├── coder.yaml        # Primary development
├── reviewer.yaml     # Code review
├── docs.yaml         # Documentation
├── debugger.yaml     # Debugging
└── tester.yaml       # Testing
```

Each agent can have its own prompt, tools, and capabilities optimized for its specific purpose.

## Future Enhancements

Planned features for agent configuration:

- Runtime agent switching during conversations
- Agent pipelines (one agent's output becomes another's input)
- Provider/model override per agent
- Agent templates and presets
- Agent behavior customization (temperature, max tokens, etc.)
