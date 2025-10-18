# Tulpa

<p align="center">
    <a href="https://github.com/tulpa-code/tulpa/releases"><img src="https://img.shields.io/github/release/tulpa-code/tulpa" alt="Latest Release"></a>
    <a href="https://github.com/tulpa-code/tulpa/actions"><img src="https://github.com/tulpa-code/tulpa/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
</p>

<p align="center">Advanced AI coding assistant with flexible agents configuration and switching for your terminal.</p>

<p align="center">Your new coding bestie, now available in your favourite terminal.<br />Your tools, your code, and your workflows, wired into your LLM of choice.</p>
<p align="center">你的新编程伙伴，现在就在你最爱的终端中。<br />你的工具、代码和工作流，都与您选择的 LLM 模型紧密相连。</p>

<p align="center"><img width="800" alt="Tulpa Demo" src="https://github.com/user-attachments/assets/58280caf-851b-470a-b6f7-d5c4ea8a1968" /></p>

---

## Philosophy

In Tibetan Buddhism, a **tulpa** is a being created through focused thought and meditation - a consciousness born from mind itself. Similarly, large language models are like captured fragments of human consciousness, crystallized into digital form.

Tulpa embraces this concept by allowing you to create as much agents and subagents as you want, combining their communication and collaboration rules in different and flexible manner.

## Features

- **Multi-Model:** choose from a wide range of LLMs or add your own via OpenAI- or Anthropic-compatible APIs
- **Flexible:** switch LLMs mid-session while preserving context
- **Session-Based:** maintain multiple work sessions and contexts per project
- **LSP-Enhanced:** Tulpa uses LSPs for additional context, just like you do
- **Extensible:** add capabilities via MCPs (`http`, `stdio`, and `sse`)
- **Works Everywhere:** first-class support in every terminal on macOS, Linux, Windows (PowerShell and WSL), FreeBSD, OpenBSD, and NetBSD

## Vision

- **Flexible Agent Switching**: Seamlessly switch between different AI models and configurations mid-conversation
- **Specialized agents**: Configure different agents for different tasks - coding, reviewing, debugging, documentation
- **Runtime MCP Management**: Dynamically enable/disable Model Context Protocol servers without restarting
- **Advanced Configuration**: Deep customization of AI behavior, context, and capabilities

## Installation

Use a package manager:

```bash
# Go install (recommended for now)
go install github.com/tulpa-code/tulpa@latest

# Homebrew (future)
brew install tulpa-code/tap/tulpa

# NPM (future)  
npm install -g @tulpa-code/tulpa

# Arch Linux (btw)
yay -S tulpa-bin

# Nix
nix run github:tulpa-code/tulpa
```

Windows users:

```bash
# Winget
winget install tulpa-code.tulpa

# Scoop
scoop bucket add tulpa https://github.com/tulpa-code/scoop-bucket.git
scoop install tulpa
```

Or, download it:

- [Packages][releases] are available in Debian and RPM formats
- [Binaries][releases] are available for Linux, macOS, Windows, FreeBSD, OpenBSD, and NetBSD

[releases]: https://github.com/tulpa-code/tulpa/releases

Or just install it with Go:

```bash
go install github.com/tulpa-code/tulpa@latest
```

## Quick Start

```bash
# Start Tulpa
tulpa

# Start with a specific provider
tulpa --provider openai
tulpa --provider anthropic

# See all options
tulpa --help
```

## Configuration

Tulpa can be configured via:

- Command-line flags
- Environment variables  
- Configuration file (`~/.config/tulpa/config.json`)

### Example Configuration

{
  "provider": {
    "id": "openai",
    "api_key": "your-api-key-here"
  },
  "options": {
    "theme": "dark",
    "model": "gpt-4"
  }
}

## Built on Crush

Tulpa extends [Charmbracelet Crush](https://github.com/charmbracelet/crush)
with multi-agent orchestration and project memory capabilities.

**Original Project**: Crush by Charmbracelet  
**License**: FSL-1.1-MIT  
**Copyright**: 2025 Charmbracelet, Inc  
**Tulpa Additions**: Copyright 2025 Tulpa Contributors

_Tulpa is an independent opensource software project and is not affiliated with Charmbracelet, Inc._

## Current Status

🚧 **Early Development** - Tulpa is currently in active development, extending the **_excellent_** foundation provided by [Charmbracelet Crush](https://github.com/charmbracelet/crush).

We're building upon Crush's solid architecture to create a more flexible and configurable AI coding assistant without hardcoded prompts and rules. This is your thought-forms at the end of your fingers.

## Roadmap

- [ ] **Enhanced Agent Configuration** - Advanced settings for AI behavior and specialization
- [ ] **Dynamic Model Switching** - Commands for switching between different AI consciousnesses
- [ ] **Runtime MCP Management** - Dynamic control over Model Context Protocol servers
- [ ] **Context Specialization** - Tailored contexts and prompts for different development tasks
- [ ] **Workflow Integration** - Seamless integration with various development workflows and tools

## Getting Involved

If you find this fork of Charm interesting and useful we'd love to have you join us on this journey of making tulpa even better!

- **Star** this repository to follow our progress
- **Contribute** ideas, code, or any other insights
- **Report issues** at [github.com/tulpa-code/tulpa/issues](https://github.com/tulpa-code/tulpa/issues)

## Genuine Pinch of Inspiration

> _"The mind is everything. What you think you become."_ - Buddha

> _"Every model is a compressed representation of human thought and knowledge."_ - Andrej Karpathy (paraphrased)

## License

[FSL-1.1-MIT](LICENSE.md)

---