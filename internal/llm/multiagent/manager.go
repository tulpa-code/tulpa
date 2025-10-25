// Package multiagent provides management for multiple agents within a session.
package multiagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tulpa-code/tulpa/internal/config"
	"github.com/tulpa-code/tulpa/internal/llm/agent"
	"github.com/tulpa-code/tulpa/internal/message"
	"github.com/tulpa-code/tulpa/internal/permission"
	"github.com/tulpa-code/tulpa/internal/session"
)

var (
	ErrAgentNotFound      = errors.New("agent not found")
	ErrAgentNotAvailable = errors.New("agent not available in this session")
	ErrNoAgentsConfigured = errors.New("no agents configured")
)

// Manager handles multiple agent instances for a session
type Manager struct {
	agents       map[string]agent.Service  // agentID -> agent instance
	agentConfigs map[string]config.Agent   // agentID -> agent config
	activeAgent  string                    // Current active agent ID
	agentHistory []string                  // Ordered list of agent IDs used (for Tab cycling)
	
	sessionID   string
	sessionSvc  session.Service
	messageSvc  message.Service
	permSvc     permission.Service
	
	// For lazy initialization
	agentFactory func(context.Context, config.Agent) (agent.Service, error)
	
	mu sync.RWMutex
}

// NewManager creates a new agent manager for a session
func NewManager(
	ctx context.Context,
	sessionID string,
	agentConfigs map[string]config.Agent,
	sessionSvc session.Service,
	messageSvc message.Service,
	permSvc permission.Service,
	agentFactory func(context.Context, config.Agent) (agent.Service, error),
) (*Manager, error) {
	if len(agentConfigs) == 0 {
		return nil, ErrNoAgentsConfigured
	}

	m := &Manager{
		agents:       make(map[string]agent.Service),
		agentConfigs: agentConfigs,
		sessionID:    sessionID,
		sessionSvc:   sessionSvc,
		messageSvc:   messageSvc,
		permSvc:      permSvc,
		agentFactory: agentFactory,
	}

	// Initialize with default agent
	activeAgentID, agentHistory, err := m.loadSessionAgentState(ctx)
	if err != nil {
		// Default to "coder" if available, otherwise first available
		defaultID := "coder"
		if _, exists := agentConfigs[defaultID]; !exists {
			// Pick first available agent
			for id := range agentConfigs {
				defaultID = id
				break
			}
		}
		activeAgentID = defaultID
		agentHistory = []string{defaultID}
	}

	m.activeAgent = activeAgentID
	m.agentHistory = agentHistory

	return m, nil
}

// loadSessionAgentState loads the agent state from the session
func (m *Manager) loadSessionAgentState(ctx context.Context) (string, []string, error) {
	sess, err := m.sessionSvc.Get(ctx, m.sessionID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load session: %w", err)
	}

	return sess.ActiveAgentID, sess.AgentHistory, nil
}

// saveSessionAgentState saves the agent state to the session
func (m *Manager) saveSessionAgentState(ctx context.Context) error {
	historyJSON, err := json.Marshal(m.agentHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal agent history: %w", err)
	}

	_, err = m.sessionSvc.UpdateAgent(ctx, m.sessionID, m.activeAgent, string(historyJSON))
	if err != nil {
		return fmt.Errorf("failed to update session agent state: %w", err)
	}

	return nil
}

// SwitchAgent switches the active agent to the specified agent ID
func (m *Manager) SwitchAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if agent exists in configuration
	if _, exists := m.agentConfigs[agentID]; !exists {
		return ErrAgentNotFound
	}

	// Ensure agent instance is created (lazy initialization)
	if _, exists := m.agents[agentID]; !exists {
		// Will be created lazily on first use
	}

	// Update active agent and history
	m.activeAgent = agentID
	
	// Update history - move to end if already exists, otherwise append
	history := m.agentHistory
	for i, id := range history {
		if id == agentID {
			history = append(history[:i], history[i+1:]...)
			break
		}
	}
	history = append(history, agentID)
	m.agentHistory = history

	return nil
}

// CurrentAgent returns the currently active agent service
func (m *Manager) CurrentAgent() (agent.Service, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getAgentInstance(m.activeAgent)
}

// ActiveAgentID returns the currently active agent ID
func (m *Manager) ActiveAgentID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeAgent
}

// AvailableAgents returns the list of available agent IDs
func (m *Manager) AvailableAgents() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agentIDs := make([]string, 0, len(m.agentConfigs))
	for id := range m.agentConfigs {
		agentIDs = append(agentIDs, id)
	}
	return agentIDs
}

// AgentHistory returns the ordered list of agent IDs used in this session
func (m *Manager) AgentHistory() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to avoid mutation
	history := make([]string, len(m.agentHistory))
	copy(history, m.agentHistory)
	return history
}

// CycleNext cycles to the next agent in the available list (Tab key behavior)
func (m *Manager) CycleNext() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	available := m.getAvailableAgentIDs()
	if len(available) <= 1 {
		return nil // No other agents to cycle to
	}

	// Find current position in available list
	currentIndex := 0
	for i, id := range available {
		if id == m.activeAgent {
			currentIndex = i
			break
		}
	}

	// Cycle to next (with wrap-around)
	nextIndex := (currentIndex + 1) % len(available)
	nextAgentID := available[nextIndex]

	return m.switchAgentInternal(nextAgentID)
}

// CyclePrevious cycles to the previous agent (Shift+Tab key behavior)
func (m *Manager) CyclePrevious() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	available := m.getAvailableAgentIDs()
	if len(available) <= 1 {
		return nil // No other agents to cycle to
	}

	// Find current position in available list
	currentIndex := 0
	for i, id := range available {
		if id == m.activeAgent {
			currentIndex = i
			break
		}
	}

	// Cycle to previous (with wrap-around)
	prevIndex := (currentIndex - 1 + len(available)) % len(available)
	prevAgentID := available[prevIndex]

	return m.switchAgentInternal(prevAgentID)
}

// getAvailableAgentIDs returns agent IDs in a consistent order for cycling
func (m *Manager) getAvailableAgentIDs() []string {
	// Return agents in a consistent order: history first, then others
	available := make([]string, 0, len(m.agentConfigs))
	
	// Add agents from history order
	added := make(map[string]bool)
	for _, id := range m.agentHistory {
		if _, exists := m.agentConfigs[id]; exists {
			available = append(available, id)
			added[id] = true
		}
	}
	
	// Add remaining agents
	for id := range m.agentConfigs {
		if !added[id] {
			available = append(available, id)
		}
	}
	
	return available
}

// switchAgentInternal performs the internal switch (must be called with lock held)
func (m *Manager) switchAgentInternal(agentID string) error {
	if _, exists := m.agentConfigs[agentID]; !exists {
		return ErrAgentNotFound
	}

	m.activeAgent = agentID
	
	// Update history
	history := m.agentHistory
	for i, id := range history {
		if id == agentID {
			history = append(history[:i], history[i+1:]...)
			break
		}
	}
	history = append(history, agentID)
	m.agentHistory = history

	return nil
}

// getAgentInstance gets or creates an agent instance (must be called with lock held)
func (m *Manager) getAgentInstance(agentID string) (agent.Service, error) {
	if agt, exists := m.agents[agentID]; exists {
		return agt, nil
	}

	agentCfg, exists := m.agentConfigs[agentID]
	if !exists {
		return nil, ErrAgentNotFound
	}

	// Lazy initialization
	slog.Info("creating agent instance", "agent_id", agentID, "session_id", m.sessionID)
	agt, err := m.agentFactory(context.Background(), agentCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent %s: %w", agentID, err)
	}

	m.agents[agentID] = agt
	return agt, nil
}

// Run executes a prompt with the current active agent
func (m *Manager) Run(ctx context.Context, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	currentAgent, err := m.CurrentAgent()
	if err != nil {
		return nil, err
	}

	// Save agent state before running
	if err := m.saveSessionAgentState(ctx); err != nil {
		slog.Warn("failed to save agent state", "error", err)
	}

	// Run with current agent
	events, err := currentAgent.Run(ctx, m.sessionID, content, attachments...)
	if err != nil {
		return nil, fmt.Errorf("failed to run agent %s: %w", m.activeAgent, err)
	}

	return events, nil
}

// Cancel cancels the current agent's operation for the session
func (m *Manager) Cancel(sessionID string) {
	m.mu.RLock()
	currentAgent, err := m.getAgentInstance(m.activeAgent)
	m.mu.RUnlock()

	if err == nil {
		currentAgent.Cancel(sessionID)
	}
}

// CancelAll cancels all agent operations
func (m *Manager) CancelAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, agt := range m.agents {
		agt.CancelAll()
	}
}

// IsSessionBusy checks if the current agent is busy for the session
func (m *Manager) IsSessionBusy(sessionID string) bool {
	m.mu.RLock()
	currentAgent, err := m.getAgentInstance(m.activeAgent)
	m.mu.RUnlock()

	if err != nil {
		return false
	}
	return currentAgent.IsSessionBusy(sessionID)
}

// IsBusy checks if any agent is busy
func (m *Manager) IsBusy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, agt := range m.agents {
		if agt.IsBusy() {
			return true
		}
	}
	return false
}

// GetAgentConfig returns the configuration for a specific agent
func (m *Manager) GetAgentConfig(agentID string) (config.Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg, exists := m.agentConfigs[agentID]
	return cfg, exists
}

// Close cleans up agent instances
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CancelAll()
}