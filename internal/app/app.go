package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/tulpa-code/tulpa/internal/config"
	"github.com/tulpa-code/tulpa/internal/csync"
	"github.com/tulpa-code/tulpa/internal/db"
	"github.com/tulpa-code/tulpa/internal/history"
	"github.com/tulpa-code/tulpa/internal/llm/agent"
	"github.com/tulpa-code/tulpa/internal/llm/multiagent"
	"github.com/tulpa-code/tulpa/internal/lsp"
	"github.com/tulpa-code/tulpa/internal/message"
	"github.com/tulpa-code/tulpa/internal/permission"
	"github.com/tulpa-code/tulpa/internal/pubsub"
	"github.com/tulpa-code/tulpa/internal/session"
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service

	AgentManagers *csync.Map[string, *multiagent.Manager] // sessionID -> manager
	LSPClients    *csync.Map[string, *lsp.Client]

	// Legacy coder agent for backwards compatibility
	coderAgent agent.Service
	CoderAgent agent.Service // Public field for TUI compatibility

	config *config.Config

	serviceEventsWG *sync.WaitGroup
	eventsCtx       context.Context
	events          chan tea.Msg
	tuiWG           *sync.WaitGroup

	// global context and cleanup functions
	globalCtx    context.Context
	cleanupFuncs []func() error
}

// New initializes a new applcation instance.
func New(ctx context.Context, conn *sql.DB, cfg *config.Config) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	files := history.NewService(q, conn)
	skipPermissionsRequests := cfg.Permissions != nil && cfg.Permissions.SkipRequests
	allowedTools := []string{}
	if cfg.Permissions != nil && cfg.Permissions.AllowedTools != nil {
		allowedTools = cfg.Permissions.AllowedTools
	}

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(cfg.WorkingDir(), skipPermissionsRequests, allowedTools),
		LSPClients:  csync.NewMap[string, *lsp.Client](),
		AgentManagers: csync.NewMap[string, *multiagent.Manager](),

		globalCtx: ctx,

		config: cfg,

		events:          make(chan tea.Msg, 100),
		serviceEventsWG: &sync.WaitGroup{},
		tuiWG:           &sync.WaitGroup{},
	}

	app.setupEvents()

	// Initialize LSP clients in the background.
	app.initLSPClients(ctx)

	// cleanup database upon app shutdown
	app.cleanupFuncs = append(app.cleanupFuncs, conn.Close)

	return app, nil
}

// GetAgentManager returns agent manager for a session
func (a *App) GetAgentManager(sessionID string) (*multiagent.Manager, error) {
	manager, exists := a.AgentManagers.Get(sessionID)
	if exists {
		return manager, nil
	}

	// Create new manager for this session
	manager, err := multiagent.NewManager(
		context.Background(),
		sessionID,
		a.config.Agents,
		a.Sessions,
		a.Messages,
		a.Permissions,
		func(ctx context.Context, cfg config.Agent) (agent.Service, error) {
			return agent.NewAgent(ctx, cfg, a.Permissions, a.Sessions, a.Messages, a.History, a.LSPClients)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent manager: %w", err)
	}

	// Store manager
	a.AgentManagers.Set(sessionID, manager)
	return manager, nil
}

func (a *App) SwitchAgent(sessionID, agentID string) error {
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return err
	}
	return manager.SwitchAgent(agentID)
}

func (a *App) RunAgent(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return nil, err
	}
	return manager.Run(ctx, content, attachments...)
}

func (a *App) ActiveAgentID(sessionID string) (string, error) {
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return "", err
	}
	return manager.ActiveAgentID(), nil
}

func (a *App) CancelAgent(sessionID string) {
	manager, exists := a.AgentManagers.Get(sessionID)
	if exists {
		manager.Cancel(sessionID)
	}
}

func (a *App) CycleNextAgent(sessionID string) error {
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return err
	}
	return manager.CycleNext()
}

func (a *App) CyclePreviousAgent(sessionID string) error {
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return err
	}
	return manager.CyclePrevious()
}

// setupEvents sets up event handlers for the app.
func (a *App) setupEvents() {
	// We can add general event handling here if needed
}




// Config returns the application configuration.
func (a *App) Config() *config.Config {
	return a.config
}

// Subscribe sends events to the TUI as tea.Msgs.
func (a *App) Subscribe(program *tea.Program) {
	defer func() {
		if r := recover(); r != nil {
			slog.Info("TUI subscription panic: attempting graceful shutdown")
			program.Quit()
		}
	}()

	a.tuiWG.Add(1)
	tuiCtx, tuiCancel := context.WithCancel(a.globalCtx)
	a.cleanupFuncs = append(a.cleanupFuncs, func() error {
		slog.Debug("Cancelling TUI message handler")
		tuiCancel()
		a.tuiWG.Wait()
		return nil
	})
	defer a.tuiWG.Done()

	for {
		select {
		case <-tuiCtx.Done():
			slog.Debug("TUI message handler shutting down")
			return
		case msg, ok := <-a.events:
			if !ok {
				slog.Debug("TUI message channel closed")
				return
			}
			program.Send(msg)
		}
	}
}

// Shutdown performs a graceful shutdown of the application.
func (a *App) Shutdown() {
	// Cancel all agent managers
	for sessionID, manager := range a.AgentManagers.Seq2() {
		manager.Cancel(sessionID)
	}

	// Shutdown all LSP clients.
	for name, client := range a.LSPClients.Seq2() {
		shutdownCtx, cancel := context.WithTimeout(a.globalCtx, 5*time.Second)
		if err := client.Close(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown LSP client", "name", name, "error", err)
		}
		cancel()
	}

	// Call cleanup functions.
	for _, cleanup := range a.cleanupFuncs {
		if cleanup != nil {
			if err := cleanup(); err != nil {
				slog.Error("Failed to cleanup app properly on shutdown", "error", err)
			}
		}
	}
}

// RunNonInteractive runs the agent in non-interactive mode.
func (a *App) RunNonInteractive(ctx context.Context, prompt string, quiet bool) error {
	// Create a temporary session for this interaction
	sess, err := a.Sessions.Create(ctx, "Non-interactive Session")
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Get the default agent manager for this session
	manager, err := a.GetAgentManager(sess.ID)
	if err != nil {
		return fmt.Errorf("failed to get agent manager: %w", err)
	}

	// Run the agent
	events, err := manager.Run(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to run agent: %w", err)
	}

	// Process events
	for event := range events {
		switch event.Type {
		case agent.AgentEventTypeResponse:
			if !quiet {
				fmt.Print(event.Message.Content())
			}
		case agent.AgentEventTypeError:
			return fmt.Errorf("agent error: %w", event.Error)
		}
	}

	return nil
}

// InitCoderAgent initializes the coder agent for backwards compatibility.
func (a *App) InitCoderAgent() error {
	if a.coderAgent != nil {
		return nil // Already initialized
	}
	
	// Create a dummy session ID for backwards compatibility
	sessionID := "legacy-coder"
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get agent manager: %w", err)
	}

	// Get active agent from manager
	agentID := manager.ActiveAgentID()
	if agentID == "" {
		// Try to get "coder" agent as default
		if err := manager.SwitchAgent("coder"); err != nil {
			return fmt.Errorf("failed to switch to coder agent: %w", err)
		}
		agentID = "coder"
	}

	// Verify agent was actually loaded
	if agentID == "" {
		return fmt.Errorf("no agent could be activated")
	}

	// Create a wrapper that delegates to manager
	wrapper := &legacyAgentWrapper{manager: manager, sessionID: sessionID}
	a.coderAgent = wrapper
	a.CoderAgent = wrapper
	return nil
}
// UpdateAgentModel updates model for legacy coder agent.
func (a *App) UpdateAgentModel() error {
	// This method is kept for compatibility but doesn't need to do anything
	// since models are handled by individual agents in new architecture
	return nil
}

// CoderAgent returns a legacy interface for compatibility with existing TUI code.
// This creates a single agent manager for backwards compatibility.
func (a *App) GetCoderAgent() agent.Service {
	if a.CoderAgent != nil {
		return a.CoderAgent // Already initialized
	}
	
	// Create a dummy session ID for backwards compatibility
	sessionID := "legacy-coder"
	manager, err := a.GetAgentManager(sessionID)
	if err != nil {
		return nil
	}

	// Get active agent from manager
	agentID := manager.ActiveAgentID()
	if agentID == "" {
		// Try to get "coder" agent as default
		if err := manager.SwitchAgent("coder"); err != nil {
			return nil
		}
	}

	// Create a wrapper that delegates to manager
	wrapper := &legacyAgentWrapper{manager: manager, sessionID: sessionID}
	a.CoderAgent = wrapper
	return wrapper
}
// legacyAgentWrapper provides backwards compatibility with the old single-agent interface
type legacyAgentWrapper struct {
	manager   *multiagent.Manager
	sessionID string
}

func (w *legacyAgentWrapper) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	return w.manager.Run(ctx, content, attachments...)
}

func (w *legacyAgentWrapper) Cancel(sessionID string) {
	w.manager.Cancel(w.sessionID)
}

func (w *legacyAgentWrapper) CancelAll() {
	w.manager.Cancel(w.sessionID)
}

func (w *legacyAgentWrapper) Subscribe(ctx context.Context) <-chan pubsub.Event[agent.AgentEvent] {
	// Get the active agent and forward its events
	activeAgent, err := w.manager.CurrentAgent()
	if err != nil || activeAgent == nil {
		// Return a closed channel if no active agent
		ch := make(chan pubsub.Event[agent.AgentEvent])
		close(ch)
		return ch
	}
	
	// Forward events from the active agent
	return activeAgent.Subscribe(ctx)
}
func (w *legacyAgentWrapper) UpdateModel() error {
	// This method may not be needed in the new architecture
	return nil
}

func (w *legacyAgentWrapper) Model() catwalk.Model {
	// Get model from active agent
	activeAgent, err := w.manager.CurrentAgent()
	if err != nil || activeAgent == nil {
		return catwalk.Model{}
	}
	return activeAgent.Model()
}

func (w *legacyAgentWrapper) IsBusy() bool {
	if w.manager == nil {
		return false
	}
	return w.manager.IsBusy()
}

func (w *legacyAgentWrapper) IsSessionBusy(sessionID string) bool {
	if w.manager == nil {
		return false
	}
	return w.manager.IsBusy()
}

func (w *legacyAgentWrapper) Summarize(ctx context.Context, sessionID string) error {
	// This would need to be implemented properly
	return fmt.Errorf("summarize not implemented in legacy wrapper")
}

func (w *legacyAgentWrapper) QueuedPrompts(sessionID string) int {
	// Return 0 for now - this would need to be implemented properly
	return 0
}

func (w *legacyAgentWrapper) ClearQueue(sessionID string) {
	// This would need to be implemented properly
}