# Multi-Agent Switching Implementation Plan

**Version**: 1.0
**Date**: 2025-10-24
**Status**: Proposal

## Executive Summary

This document outlines the implementation plan for introducing **runtime agent switching** in Tulpa, allowing users to cycle between multiple primary agents using keyboard shortcuts while maintaining shared session context. This feature draws inspiration from OpenCode's agent switching mechanism while adapting it to Tulpa's architecture.

### Complexity Assessment: **MEDIUM-HIGH** ⚠️

**Estimated Effort**: 2-3 weeks for experienced Go developer
**Risk Level**: Medium (requires careful state management and UI refactoring)
**Breaking Changes**: Yes (keybinding changes, session schema modifications)

---

## Table of Contents

1. [Requirements Analysis](#requirements-analysis)
2. [Current Architecture Analysis](#current-architecture-analysis)
3. [Research Findings: OpenCode Agent Switching](#research-findings-opencode-agent-switching)
4. [Proposed Architecture](#proposed-architecture)
5. [Implementation Phases](#implementation-phases)
6. [Technical Challenges](#technical-challenges)
7. [Testing Strategy](#testing-strategy)
8. [Migration Path](#migration-path)
9. [Risks and Mitigations](#risks-and-mitigations)
10. [Future Enhancements](#future-enhancements)

---

## 1. Requirements Analysis

### 1.1 Core Requirements

| Req # | Requirement | Complexity | Priority |
|-------|-------------|------------|----------|
| R1 | **Agent Switching via Tab Key**: Cycle through primary agents using Tab, remap chat focus to Shift+Tab | Medium | P0 |
| R2 | **Shared Session Context**: All agents in a session share the same message history and context | High | P0 |
| R3 | **Configurable Subagents**: Primary agents specify which subagents they can invoke via agent tool | Medium | P1 |
| R4 | **Visual Indicator**: Display active agent name/identifier in status bar | Low | P0 |
| R5 | **Per-Agent Prompts**: Each agent uses its own system prompt when generating responses | Low | P0 |
| R6 | **Per-Agent Tool Access**: Agents maintain their configured tool restrictions | Low | P0 |

### 1.2 User Workflow

```
User Flow:
1. User starts session → Default primary agent (e.g., "coder") is active
2. User types message → Coder agent responds with its prompt/tools
3. User presses Tab → Switches to next primary agent (e.g., "reviewer")
4. User types message → Reviewer agent responds with ITS prompt/tools
5. Tab cycles through: coder → reviewer → docs → [back to coder]
6. Shift+Tab for reverse cycle
7. All agents see same conversation history
```

### 1.3 Out of Scope

- Multi-agent collaboration (agents talking to each other automatically)
- Parallel agent execution (multiple agents responding to same prompt)
- Agent-specific conversation branching
- Custom agent creation from UI (YAML editing only)

---

## 2. Current Architecture Analysis

### 2.1 Current Agent System

**Current State**:
```go
// Single agent instance per session
type agent struct {
    agentCfg    config.Agent      // Fixed at initialization
    provider    provider.Provider  // Fixed model/provider
    // ... other fields
}

// Agent creation in app startup
coderAgent := agent.NewAgent(ctx, cfg.Agents["coder"], ...)
```

**Limitations**:
- ❌ One agent per App instance (hardcoded "coder")
- ❌ No runtime agent switching
- ❌ Agent tied to App lifecycle, not session lifecycle
- ✅ Subagent (task) already works via agent tool
- ✅ Sessions track ParentSessionID

### 2.2 Current Session Structure

```go
type Session struct {
    ID               string
    ParentSessionID  string
    Title            string
    MessageCount     int64
    PromptTokens     int64
    CompletionTokens int64
    SummaryMessageID string
    Cost             float64
    CreatedAt        int64
    UpdatedAt        int64
}
```

**Missing**:
- No `ActiveAgentID` field
- No `AgentHistory` tracking (which agents were used when)

### 2.3 Current Message Structure

```go
type Message struct {
    ID        string
    SessionID string
    Role      Role  // User, Assistant, Tool
    Parts     []ContentPart
    Model     string    // Model ID used
    Provider  string    // Provider ID
    // ... metadata
}
```

**Good**: Already tracks which model/provider generated each message
**Missing**: No direct `AgentID` field (currently inferred from model/provider)

### 2.4 Current TUI Keybindings

```go
// chat/keys.go
Tab: key.NewBinding(
    key.WithKeys("tab"),
    key.WithHelp("tab", "change focus"),  // Currently: Editor ↔ Chat ↔ Sidebar
)
```

**Conflict**: Tab is already used for focus cycling
**Solution**: Remap to Shift+Tab (OpenCode uses Tab for agent switch)

### 2.5 Current Agent Tool (Subagent Invocation)

```go
// agent-tool.go
func (b *agentTool) Run(ctx context.Context, call tools.ToolCall) {
    // Creates NEW session with toolCallID
    session := b.sessions.CreateTaskSession(ctx, call.ID, sessionID, "New Agent Session")

    // Runs task agent in SEPARATE session
    result := b.agent.Run(ctx, session.ID, params.Prompt)

    // Returns text response
    return tools.NewTextResponse(response.Content().String())
}
```

**Current Behavior**: Subagent runs in isolated session (NOT shared context)
**Required Change**: Subagent should run in SAME session but with different agent config

---

## 3. Research Findings: OpenCode Agent Switching

### 3.1 OpenCode Agent Model

Based on research and documentation analysis:

```yaml
# OpenCode-style agent configuration
agents:
  - id: coder
    name: "Coder"
    model: claude-sonnet-3-5
    tools: [all]

  - id: reviewer
    name: "Code Reviewer"
    model: gpt-4
    tools: [view, grep, glob]  # Read-only

  - id: docs
    name: "Documentation Writer"
    model: claude-sonnet-3-5
    tools: [view, edit, write]
```

**Agent Switching**:
- **Tab key** cycles through primary agents
- **Ctrl+Left/Right** navigates between subagent tabs (when spawned)
- All agents in a session share the **same conversation history**
- Each agent applies its own **system prompt** when generating responses
- Visual indicator shows **active agent** in status bar

### 3.2 Context Sharing Mechanism

**Key Insight**: OpenCode doesn't create separate sessions for different agents. Instead:

1. **Single Session** = One conversation thread
2. **Multiple Agents** = Different "personalities" contributing to same thread
3. **Message History** = Shared across all agents
4. **System Prompt** = Swapped when agent changes
5. **Tool Access** = Restricted per agent config

**Example**:
```
Session #abc123:
  Message 1: [User] "What's the bug in auth.go?"
  Message 2: [Assistant - Coder Agent] "Let me check..."
  <User presses Tab to switch to Reviewer>
  Message 3: [User] "Review the fix I made"
  Message 4: [Assistant - Reviewer Agent] "The fix looks good, but..."
```

Both agents see Messages 1-2, but respond with their own prompts/tools.

### 3.3 OpenCode Implementation Patterns

From architectural analysis (inferred):

```go
// Conceptual OpenCode pattern
type Session struct {
    ID            string
    ActiveAgentID string    // Which agent is currently active
    Messages      []Message
}

type Message struct {
    AgentID  string  // Which agent generated this
    Content  string
    Model    string
}

// On Tab press:
func (s *Session) SwitchAgent(newAgentID string) {
    s.ActiveAgentID = newAgentID
    // Next message will use new agent's config
}
```

---

## 4. Proposed Architecture

### 4.1 New Data Structures

#### 4.1.1 Enhanced Session Schema

```go
type Session struct {
    // Existing fields
    ID               string
    ParentSessionID  string
    Title            string
    MessageCount     int64
    PromptTokens     int64
    CompletionTokens int64
    SummaryMessageID string
    Cost             float64
    CreatedAt        int64
    UpdatedAt        int64

    // NEW FIELDS
    ActiveAgentID    string   // Currently active primary agent
    AgentHistory     []string // Ordered list of agent IDs used (for Tab cycling)
}
```

**Database Migration**:
```sql
ALTER TABLE sessions ADD COLUMN active_agent_id TEXT NOT NULL DEFAULT 'coder';
ALTER TABLE sessions ADD COLUMN agent_history TEXT NOT NULL DEFAULT '[]'; -- JSON array
```

#### 4.1.2 Enhanced Message Schema

```go
type Message struct {
    // Existing fields
    ID        string
    SessionID string
    Role      Role
    Parts     []ContentPart
    Model     string
    Provider  string
    CreatedAt int64
    UpdatedAt int64

    // NEW FIELD
    AgentID   string  // Which agent generated this assistant message
}
```

**Database Migration**:
```sql
ALTER TABLE messages ADD COLUMN agent_id TEXT; -- Nullable for backward compat
```

#### 4.1.3 Multi-Agent Manager

**New Service**: `internal/llm/multiagent/manager.go`

```go
package multiagent

// Manager handles multiple agent instances for a session
type Manager struct {
    agents       map[string]agent.Service  // agentID → agent instance
    activeAgent  string                     // Current active agent ID
    sessionSvc   session.Service
    messageSvc   message.Service
}

// Create agent manager for a session
func NewManager(
    ctx context.Context,
    sessionID string,
    agentConfigs map[string]config.Agent,
    // ... dependencies
) (*Manager, error)

// Switch active agent
func (m *Manager) SwitchAgent(agentID string) error

// Get current agent
func (m *Manager) CurrentAgent() agent.Service

// List available agents
func (m *Manager) AvailableAgents() []string

// Run message with current agent
func (m *Manager) Run(ctx context.Context, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error)

// Cycle to next agent (Tab key)
func (m *Manager) CycleNext() error

// Cycle to previous agent (Shift+Tab if desired)
func (m *Manager) CyclePrevious() error
```

### 4.2 Modified App Structure

```go
// internal/app/app.go
type App struct {
    // OLD: Single agent
    // agent agent.Service

    // NEW: Multi-agent manager per session
    agentManagers map[string]*multiagent.Manager  // sessionID → manager
    agentManagersMu sync.RWMutex

    // ... other fields
}

// Get or create agent manager for session
func (a *App) GetAgentManager(sessionID string) (*multiagent.Manager, error)

// Switch agent in session
func (a *App) SwitchAgent(sessionID, agentID string) error
```

### 4.3 Modified Agent Tool (Subagent)

```go
// internal/llm/agent/agent-tool.go
func (b *agentTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
    // Get parent session's agent config
    parentSession := // ... get from context
    primaryAgent := parentSession.ActiveAgentID

    // Get allowed subagents for this primary agent
    allowedSubagents := b.agentCfg.AllowedSubagents // NEW config field

    // Determine which subagent to use (from params or config)
    subagentID := determineSubagent(params, allowedSubagents)

    // Run subagent in SAME session, but with different agent config
    // Create temporary "view" of session with subagent active
    result := b.runSubagent(ctx, sessionID, subagentID, params.Prompt)

    return tools.NewTextResponse(result)
}
```

### 4.4 Enhanced Agent YAML Config

```yaml
# ~/.config/tulpa/agents/coder.yaml
id: coder
name: Coder
description: Primary coding agent
prompt: |
  You are Tulpa, a coding assistant...

model:
  type: large

tools:
  allowed:
    - bash
    - edit
    - write
    - view
    - grep
    - glob
    - agent  # Can invoke subagents

# NEW: Subagent configuration
subagents:
  allowed:
    - task      # Can invoke task agent
    - analyzer  # Can invoke analyzer agent
  default: task # Default when not specified

mcp:
  allowed:
    server1: []

lsp:
  allowed:
    - gopls
```

```yaml
# ~/.config/tulpa/agents/reviewer.yaml
id: reviewer
name: Code Reviewer
description: Reviews code for quality
prompt: |
  You are a code reviewer focused on...

model:
  type: large

tools:
  allowed:
    - view
    - grep
    - glob
    # NO edit, write, bash (read-only)

subagents:
  allowed: []  # Reviewers don't spawn subagents
```

### 4.5 TUI Changes

#### 4.5.1 New Keybindings

```go
// internal/tui/page/chat/keys.go
type KeyMap struct {
    NewSession       key.Binding
    AddAttachment    key.Binding
    Cancel           key.Binding
    SwitchAgent      key.Binding  // NEW: Tab
    ChangeFocus      key.Binding  // NEW: Shift+Tab (old Tab functionality)
    Details          key.Binding
}

func DefaultKeyMap() KeyMap {
    return KeyMap{
        // ... existing
        SwitchAgent: key.NewBinding(
            key.WithKeys("tab"),
            key.WithHelp("tab", "switch agent"),
        ),
        ChangeFocus: key.NewBinding(
            key.WithKeys("shift+tab"),
            key.WithHelp("shift+tab", "change focus"),
        ),
    }
}
```

#### 4.5.2 Status Bar Update

```go
// internal/tui/components/core/status/status.go
type StatusCmp struct {
    // ... existing fields
    activeAgent string  // NEW: Display active agent
}

// Render example: "coder │ Session: My Project │ Cost: $0.45"
func (s *StatusCmp) View() string {
    agentIndicator := lipgloss.NewStyle().
        Foreground(lipgloss.Color("42")).
        Bold(true).
        Render(s.activeAgent)

    return fmt.Sprintf("%s │ %s", agentIndicator, s.otherInfo)
}
```

#### 4.5.3 Chat Page Agent Switching

```go
// internal/tui/page/chat/chat.go
func (c *ChatPage) Update(msg tea.Msg) (util.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle Tab for agent switching
        if key.Matches(msg, c.keys.SwitchAgent) {
            return c, c.switchToNextAgent()
        }

        // Handle Shift+Tab for focus changing
        if key.Matches(msg, c.keys.ChangeFocus) {
            return c, c.changeFocus()
        }
    }
}

func (c *ChatPage) switchToNextAgent() tea.Cmd {
    return func() tea.Msg {
        // Get agent manager for current session
        mgr := c.app.GetAgentManager(c.sessionID)

        // Cycle to next agent
        err := mgr.CycleNext()
        if err != nil {
            return ErrorMsg{err}
        }

        // Update status bar
        return AgentSwitchedMsg{
            AgentID: mgr.CurrentAgent().ID,
        }
    }
}
```

---

## 5. Implementation Phases

### Phase 1: Foundation (Week 1)

**Goal**: Set up multi-agent infrastructure without breaking existing functionality

**Tasks**:
1. ✅ Database migrations for Session and Message schema
2. ✅ Create `internal/llm/multiagent/manager.go`
3. ✅ Update `config.Agent` to support `AllowedSubagents` field
4. ✅ Modify App to support multi-agent managers
5. ✅ Write unit tests for agent manager

**Deliverables**:
- Multi-agent manager with basic agent switching
- Database schema updated
- Tests passing

**Risk**: Low (additive changes, backward compatible)

### Phase 2: TUI Integration (Week 2, Days 1-3)

**Goal**: Integrate agent switching into UI with Tab key

**Tasks**:
1. ✅ Update keybindings (Tab → agent switch, Shift+Tab → focus)
2. ✅ Add agent indicator to status bar
3. ✅ Handle Tab key in chat page
4. ✅ Update help text
5. ✅ Test keyboard navigation thoroughly

**Deliverables**:
- Tab key cycles through agents
- Status bar shows active agent
- Shift+Tab changes focus

**Risk**: Medium (keybinding conflicts, UX changes)

### Phase 3: Context Sharing (Week 2, Days 4-5)

**Goal**: Ensure all agents in a session share message history

**Tasks**:
1. ✅ Modify agent.Run() to use session's message history (already done)
2. ✅ Track AgentID in message creation
3. ✅ Update message rendering to show which agent responded
4. ✅ Test context continuity across agent switches

**Deliverables**:
- Agents see all previous messages
- Each message tagged with generating agent
- Conversation flows naturally across switches

**Risk**: Low (mostly using existing session/message infrastructure)

### Phase 4: Subagent Configuration (Week 3)

**Goal**: Primary agents configure which subagents they can invoke

**Tasks**:
1. ✅ Extend YAML schema for `subagents.allowed`
2. ✅ Update agent tool to check allowed subagents
3. ✅ Error handling when disallowed subagent requested
4. ✅ Update documentation with subagent examples

**Deliverables**:
- YAML configs specify allowed subagents
- Agent tool validates subagent access
- Error messages guide users

**Risk**: Low (extension of existing agent tool)

### Phase 5: Polish & Testing (Week 3, Days 4-5)

**Goal**: End-to-end testing and documentation

**Tasks**:
1. ✅ Integration tests for multi-agent workflows
2. ✅ Update user documentation
3. ✅ Create example agent configs (reviewer, docs, debugger)
4. ✅ Performance testing (agent switching latency)
5. ✅ Migration guide for existing users

**Deliverables**:
- Comprehensive test suite
- User guide with examples
- Migration documentation

**Risk**: Low (polish phase)

---

## 6. Technical Challenges

### 6.1 Challenge: Shared Context State Management

**Problem**: Multiple agent instances need to read the same message history but apply different prompts.

**Current State**:
```go
// Agent's Run() method gets messages
msgs, err := a.messages.List(ctx, sessionID)
// Applies single system prompt
response := a.provider.StreamResponse(ctx, msgHistory, tools)
```

**Solution**:
```go
// Manager handles context
func (m *Manager) Run(ctx, sessionID, content) {
    currentAgent := m.agents[m.activeAgent]

    // Get shared message history
    msgs := m.messageSvc.List(ctx, sessionID)

    // Current agent applies ITS prompt
    currentAgent.Run(ctx, sessionID, content)
    // Agent internally uses its configured system prompt
}
```

**Complexity**: **Low** - Agents already retrieve messages per session. Just need to ensure agents use their own prompts, which is already the case.

---

### 6.2 Challenge: Agent Initialization Performance

**Problem**: Creating all agents upfront for every session could be slow.

**Current**:
- 1 agent instance for entire app (fast)

**Proposed**:
- N agents × M sessions = N×M instances (potentially slow)

**Solution 1: Lazy Initialization**
```go
type Manager struct {
    agentConfigs map[string]config.Agent  // Configs (cheap)
    agents       map[string]agent.Service // Instances (expensive, lazy)
}

func (m *Manager) SwitchAgent(agentID string) error {
    // Create agent on first use
    if _, exists := m.agents[agentID]; !exists {
        m.agents[agentID] = agent.NewAgent(ctx, m.agentConfigs[agentID], ...)
    }
    m.activeAgent = agentID
}
```

**Solution 2: Agent Pooling** (if needed)
```go
// Global pool, shared across sessions
type AgentPool struct {
    agents map[string]agent.Service  // Shared instances
}

// Sessions borrow agents from pool
func (p *AgentPool) Get(agentID string) agent.Service
```

**Recommendation**: Start with Solution 1 (lazy init). Add pooling only if performance issues arise.

**Complexity**: **Medium**

---

### 6.3 Challenge: Keybinding Conflict

**Problem**: Tab is currently used for focus cycling (Editor ↔ Chat ↔ Sidebar).

**Current Users**: Muscle memory for Tab to switch focus.

**Solution**:
1. **Remap to Shift+Tab** for focus (as proposed)
2. **Add migration notice** in release notes
3. **Make configurable** (optional enhancement)

**UX Consideration**:
- OpenCode uses Tab for agent switch → Good precedent
- Shift+Tab is a common pattern (reverse tab in browsers)
- Document prominently in changelog

**Complexity**: **Low** (implementation) / **Medium** (user communication)

---

### 6.4 Challenge: Subagent Context Isolation

**Problem**: Current subagent (task agent) runs in SEPARATE session. Need to run in SAME session.

**Current**:
```go
// Creates new session
session := b.sessions.CreateTaskSession(ctx, call.ID, sessionID, "Task")
result := b.agent.Run(ctx, session.ID, prompt)
```

**Proposed**:
```go
// Same session, different agent
func (b *agentTool) Run(ctx, call) {
    // Use PARENT session
    subagentID := getSubagentID(params)

    // Temporarily switch session's active agent
    manager := b.getSessionManager(sessionID)
    manager.SwitchAgent(subagentID)

    // Run in same session
    result := manager.Run(ctx, params.Prompt)

    // Restore primary agent
    manager.SwitchAgent(primaryAgentID)

    return result
}
```

**Complexity**: **High** - Requires careful state management to avoid race conditions.

**Alternative**: Keep subagents in separate sessions (simpler, but less context sharing).

---

### 6.5 Challenge: Cost Tracking

**Problem**: Session.Cost aggregates all agents. Need per-agent breakdown (optional).

**Current**:
```go
type Session struct {
    Cost float64  // Total cost
}
```

**Enhanced** (optional):
```go
type Session struct {
    Cost      float64            // Total
    AgentCosts map[string]float64 // Per agent breakdown
}
```

**Complexity**: **Low** (if implemented)

---

## 7. Testing Strategy

### 7.1 Unit Tests

```go
// internal/llm/multiagent/manager_test.go
func TestManager_SwitchAgent(t *testing.T) {
    mgr := setupManager(t, []string{"coder", "reviewer"})

    // Start with coder
    assert.Equal(t, "coder", mgr.CurrentAgent().ID)

    // Switch to reviewer
    err := mgr.SwitchAgent("reviewer")
    assert.NoError(t, err)
    assert.Equal(t, "reviewer", mgr.CurrentAgent().ID)
}

func TestManager_CycleNext(t *testing.T) {
    mgr := setupManager(t, []string{"coder", "reviewer", "docs"})

    mgr.CycleNext()
    assert.Equal(t, "reviewer", mgr.CurrentAgent().ID)

    mgr.CycleNext()
    assert.Equal(t, "docs", mgr.CurrentAgent().ID)

    mgr.CycleNext()
    assert.Equal(t, "coder", mgr.CurrentAgent().ID) // Wraps
}
```

### 7.2 Integration Tests

```go
// internal/app/multiagent_integration_test.go
func TestMultiAgentConversation(t *testing.T) {
    app := setupTestApp(t)

    // Create session
    session := app.CreateSession("Test")

    // Message 1: Coder agent
    app.SendMessage(session.ID, "Write a function")
    // Assert: Response from coder agent

    // Switch to reviewer
    app.SwitchAgent(session.ID, "reviewer")

    // Message 2: Reviewer agent
    app.SendMessage(session.ID, "Review the code")
    // Assert: Response from reviewer agent
    // Assert: Reviewer sees message 1 in context
}
```

### 7.3 TUI Tests

```go
// internal/tui/page/chat/agent_switching_test.go
func TestChatPage_TabSwitchesAgent(t *testing.T) {
    model := setupChatPage(t)

    // Simulate Tab key
    model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})

    // Assert: Agent switched in status bar
    // Assert: Editor still focused
}
```

---

## 8. Migration Path

### 8.1 Database Migration

```sql
-- Migration: 0005_multi_agent_support.sql
BEGIN TRANSACTION;

-- Add agent fields to sessions
ALTER TABLE sessions
    ADD COLUMN active_agent_id TEXT NOT NULL DEFAULT 'coder';

ALTER TABLE sessions
    ADD COLUMN agent_history TEXT NOT NULL DEFAULT '[]';

-- Add agent field to messages
ALTER TABLE messages
    ADD COLUMN agent_id TEXT;

-- Backfill existing messages (assume all were from coder)
UPDATE messages
SET agent_id = 'coder'
WHERE agent_id IS NULL AND role = 'assistant';

COMMIT;
```

### 8.2 Config Migration

**Automatic**: YAML loader already handles missing fields gracefully.

**User Action Required**: None for basic usage. Optional to add `subagents` config.

### 8.3 Breaking Changes

| Change | Impact | Mitigation |
|--------|--------|------------|
| Tab → Agent switch | Users expecting focus change | Shift+Tab for focus, document prominently |
| Session schema | Old databases incompatible | Auto-migration on startup |
| Subagent behavior | If relying on separate sessions | Document change, provide flag to opt-in to old behavior |

---

## 9. Risks and Mitigations

### 9.1 High-Risk Items

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **State race conditions** between agent switches | Medium | High | Mutex locks on manager, thorough testing |
| **Memory usage** from N×M agent instances | Low | Medium | Lazy initialization, agent pooling |
| **User confusion** from Tab key change | High | Low | Migration guide, in-app tutorial |
| **Subagent context** isolation issues | Medium | High | Keep separate sessions initially (Phase 2 feature) |

### 9.2 Medium-Risk Items

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Performance degradation** on agent switch | Low | Medium | Profile and optimize, cache provider instances |
| **Prompt mixing** between agents | Low | High | Strict prompt isolation in agent.Run(), extensive testing |
| **Cost tracking** inaccuracy | Low | Low | Verify aggregation logic, add per-agent breakdown |

---

## 10. Future Enhancements

### 10.1 Phase 2 Features (Post-MVP)

1. **Agent Collaboration**
   - Agents can explicitly invoke each other (not just via tool)
   - Example: Coder asks Reviewer for feedback inline

2. **Agent Pipelines**
   - Define workflows: Coder → Reviewer → Docs
   - Automatic handoff based on rules

3. **Configurable Keybindings**
   - Let users choose Tab vs other keys
   - Per-agent keybindings (e.g., Ctrl+1 for coder, Ctrl+2 for reviewer)

4. **Agent Avatars/Icons**
   - Visual indicators beyond text (color coding, icons)

5. **Agent-Specific Message Filtering**
   - View conversation as seen by specific agent
   - "Show me only coder's messages"

6. **Sub-Session Context**
   - Subagents optionally run in separate context but visible to user
   - Navigate between main thread and subagent threads

### 10.2 Long-Term Vision

- **GUI for Agent Creation**: Web UI to design agents without YAML
- **Agent Marketplace**: Share/download community agents
- **Multi-Modal Agents**: Different models for different parts (vision, code, etc.)
- **Adaptive Routing**: AI determines which agent to use for a task

---

## Appendix A: Code Structure Summary

```
tulpa/
├── internal/
│   ├── app/
│   │   └── app.go                      # MODIFIED: Multi-agent manager integration
│   ├── llm/
│   │   ├── agent/
│   │   │   ├── agent.go                # MODIFIED: Per-agent prompt application
│   │   │   └── agent-tool.go           # MODIFIED: Subagent validation
│   │   └── multiagent/                 # NEW PACKAGE
│   │       ├── manager.go              # Agent lifecycle management
│   │       ├── manager_test.go
│   │       └── cycling.go              # Tab cycling logic
│   ├── session/
│   │   └── session.go                  # MODIFIED: Add ActiveAgentID, AgentHistory
│   ├── message/
│   │   └── message.go                  # MODIFIED: Add AgentID field
│   ├── tui/
│   │   ├── tui.go                      # MODIFIED: Handle agent switching
│   │   ├── page/chat/
│   │   │   ├── chat.go                 # MODIFIED: Tab/Shift+Tab handling
│   │   │   └── keys.go                 # MODIFIED: New keybindings
│   │   └── components/core/status/
│   │       └── status.go               # MODIFIED: Show active agent
│   └── config/
│       ├── agent_config.go             # MODIFIED: Add AllowedSubagents
│       └── config.go
└── docs/
    ├── AGENT_CONFIGURATION.md          # MODIFIED: Multi-agent examples
    └── MULTI_AGENT_GUIDE.md            # NEW: User guide
```

---

## Appendix B: Example Workflows

### Workflow 1: Code Review Cycle

```
1. User: "Implement user authentication"
   Agent: [Coder] "I'll create the auth system..."
   → Coder writes auth code

2. [User presses Tab → switches to Reviewer]
   Status: "reviewer │ ..."

3. User: "Review this implementation"
   Agent: [Reviewer] "Looking at the code..."
   → Reviewer provides feedback (read-only, can't edit)

4. [User presses Tab → back to Coder]
   Status: "coder │ ..."

5. User: "Apply the reviewer's suggestions"
   Agent: [Coder] "I'll update based on feedback..."
   → Coder makes changes (has full tool access)
```

### Workflow 2: Documentation Generation

```
1. Agent: Coder (writes feature)
2. [Tab] → Docs agent
3. User: "Document the new feature"
4. Agent: Docs (reads code, writes docs)
5. [Tab] → Coder
6. Continues development
```

---

## Appendix C: Comparison with Alternatives

### Alternative 1: No Agent Switching (Status Quo)

**Pros**: Simple, no changes needed
**Cons**: Users can't leverage specialized agents, less flexible

### Alternative 2: Separate Sessions Per Agent

**Pros**: Complete isolation, easier to implement
**Cons**: No context sharing, fragmented conversation

### Alternative 3: Agent as Tool Parameter

```
User: "@reviewer please review this code"
```

**Pros**: Explicit control, no UI changes
**Cons**: Verbose, requires typing agent name every time

### Alternative 4: Automatic Agent Selection (AI Router)

**Pros**: Hands-free, intelligent
**Cons**: Unpredictable, harder to implement, loses user control

**Recommendation**: Proceed with Tab-based switching (explicit, predictable, familiar from OpenCode).

---

## Conclusion

Implementing multi-agent switching in Tulpa is a **medium-high complexity** feature requiring:

- **3-4 weeks** development time
- Careful **state management** for shared context
- **Database schema** changes with migration
- **UX changes** with Tab key remapping

**Key Success Factors**:
1. ✅ Lazy agent initialization for performance
2. ✅ Thorough testing of context sharing
3. ✅ Clear migration documentation
4. ✅ Graceful handling of edge cases (race conditions, invalid agents)

**Recommended Approach**: Phased implementation starting with foundation (multi-agent manager), then TUI integration, then advanced features (configurable subagents).

**Risk Level**: Medium - Most components already exist (agents, sessions, messages). Main work is orchestration and UX.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-24
**Maintainer**: Tulpa Development Team
