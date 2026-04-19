package layer6

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/realityos/aizo/storage"
)

// ActionExecutor executes proposed actions
type ActionExecutor interface {
	ExecuteAction(ctx context.Context, action *ProposedAction) (*ExecutionResult, error)
}

// Manager orchestrates the rule engine, learning, and proposal lifecycle
type Manager struct {
	ruleEngine     *RuleEngine
	learningEngine *LearningEngine
	store          *SQLiteStore
	executor       ActionExecutor
	proposals      []*ActionProposal
	tuningTicker   *time.Ticker
	stopChan       chan struct{}
	mu             sync.RWMutex
}

// NewManager creates a new Layer 6 manager
func NewManager(db *storage.DB, executor ActionExecutor) *Manager {
	var store *SQLiteStore
	if db != nil {
		store = NewSQLiteStore(db.SQL())
	}

	ruleEngine := NewRuleEngine(store)
	learningEngine := NewLearningEngine(store, ruleEngine)

	// Load default rules
	for _, rule := range DefaultRules() {
		ruleEngine.AddRule(rule)
	}

	// Load user rules from ~/.aizo/rules/
	home, err := os.UserHomeDir()
	if err == nil {
		rulesDir := filepath.Join(home, ".aizo", "rules")
		if rules, err := LoadRulesFromDir(rulesDir); err == nil {
			for _, r := range rules {
				ruleEngine.AddRule(r)
			}
		}
	}

	// Load persisted rule stats from SQLite
	if store != nil {
		if stats, err := store.LoadRuleStats(); err == nil {
			for ruleID, s := range stats {
				rule := ruleEngine.GetRule(ruleID)
				if rule != nil {
					rule.SuccessCount = s.SuccessCount
					rule.FailureCount = s.FailureCount
				}
			}
		}
	}

	m := &Manager{
		ruleEngine:     ruleEngine,
		learningEngine: learningEngine,
		store:          store,
		executor:       executor,
		proposals:      make([]*ActionProposal, 0),
		stopChan:       make(chan struct{}),
	}

	return m
}

// Start begins periodic threshold tuning
func (m *Manager) Start(ctx context.Context) {
	m.tuningTicker = time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopChan:
				return
			case <-m.tuningTicker.C:
				m.learningEngine.TuneThresholds()
			}
		}
	}()
}

// Stop stops the manager
func (m *Manager) Stop() {
	if m.tuningTicker != nil {
		m.tuningTicker.Stop()
	}
	close(m.stopChan)
}

// ProcessEvent evaluates rules against a system event and queues a proposal
func (m *Manager) ProcessEvent(event *SystemEvent) (*ActionProposal, error) {
	proposal, err := m.ruleEngine.Evaluate(event)
	if err != nil || proposal == nil {
		return nil, err
	}

	m.mu.Lock()
	m.proposals = append(m.proposals, proposal)
	m.mu.Unlock()

	if m.store != nil {
		m.store.StoreProposal(proposal)
	}

	// Auto-execute if approved
	if !proposal.RequiresApproval {
		go m.executeProposal(proposal)
	}

	return proposal, nil
}

// ProcessSummary evaluates rules against a system summary
func (m *Manager) ProcessSummary(summary *SystemSummary) ([]*ActionProposal, error) {
	proposals, err := m.ruleEngine.EvaluateSummary(summary)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	for _, p := range proposals {
		m.proposals = append(m.proposals, p)
		if m.store != nil {
			m.store.StoreProposal(p)
		}
		if !p.RequiresApproval {
			go m.executeProposal(p)
		}
	}
	m.mu.Unlock()

	return proposals, nil
}

// ApproveProposal approves a pending proposal and executes it
func (m *Manager) ApproveProposal(id, approver string) error {
	m.mu.Lock()
	var found *ActionProposal
	for _, p := range m.proposals {
		if p.ID == id {
			if p.Status != ProposalPending {
				m.mu.Unlock()
				return fmt.Errorf("proposal not pending: %s", p.Status)
			}
			p.Status = ProposalApproved
			p.ApprovedBy = approver
			p.ApprovedAt = time.Now()
			found = p
			break
		}
	}
	m.mu.Unlock()

	if found == nil {
		return fmt.Errorf("proposal not found: %s", id)
	}

	if m.store != nil {
		m.store.StoreProposal(found)
	}

	go m.executeProposal(found)
	return nil
}

// RejectProposal rejects a pending proposal
func (m *Manager) RejectProposal(id, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.proposals {
		if p.ID == id {
			if p.Status != ProposalPending {
				return fmt.Errorf("proposal not pending: %s", p.Status)
			}
			p.Status = ProposalRejected
			p.Result = reason
			if m.store != nil {
				m.store.StoreProposal(p)
			}
			// Record as failure for learning
			m.learningEngine.RecordOutcome(p, false, 0)
			return nil
		}
	}
	return fmt.Errorf("proposal not found: %s", id)
}

// GetPendingProposals returns all pending proposals
func (m *Manager) GetPendingProposals() []*ActionProposal {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pending := make([]*ActionProposal, 0)
	for _, p := range m.proposals {
		if p.Status == ProposalPending {
			pending = append(pending, p)
		}
	}
	return pending
}

// GetAllProposals returns all proposals
func (m *Manager) GetAllProposals() []*ActionProposal {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ActionProposal, len(m.proposals))
	copy(result, m.proposals)
	return result
}

// AddRule adds a rule to the engine
func (m *Manager) AddRule(rule *Rule) {
	m.ruleEngine.AddRule(rule)
}

// ListRules returns all rules
func (m *Manager) ListRules() []*Rule {
	return m.ruleEngine.ListRules()
}

// TuneThresholds manually triggers threshold tuning
func (m *Manager) TuneThresholds() []string {
	return m.learningEngine.TuneThresholds()
}

// SuggestRules returns pattern-mined rule suggestions
func (m *Manager) SuggestRules() []*SuggestedRule {
	return m.learningEngine.MinePatterns()
}

// GetStats returns manager statistics
func (m *Manager) GetStats() *ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ManagerStats{
		TotalProposals: len(m.proposals),
		ByStatus:       make(map[ProposalStatus]int),
		TotalRules:     len(m.ruleEngine.ListRules()),
	}

	for _, p := range m.proposals {
		stats.ByStatus[p.Status]++
		if p.Status == ProposalPending {
			stats.PendingProposals++
		}
	}

	if m.store != nil {
		incidents, _ := m.store.LoadIncidents(1000)
		stats.TotalIncidents = len(incidents)
	}

	return stats
}

// executeProposal executes an approved or auto-approved proposal
func (m *Manager) executeProposal(proposal *ActionProposal) {
	if m.executor == nil {
		return
	}

	proposal.Status = ProposalExecuting
	start := time.Now()

	action := &ProposedAction{
		Description: proposal.Reasoning,
		ActionType:  proposal.Action,
		EntityID:    proposal.EntityID,
		Parameters:  proposal.Parameters,
		Risk:        proposal.Risk,
		Reversible:  true,
	}

	result, err := m.executor.ExecuteAction(context.Background(), action)
	duration := time.Since(start)

	proposal.ExecutedAt = time.Now()

	if err != nil || (result != nil && !result.Success) {
		proposal.Status = ProposalFailed
		if result != nil {
			proposal.Result = result.Error
		} else if err != nil {
			proposal.Result = err.Error()
		}
		m.learningEngine.RecordOutcome(proposal, false, duration)
	} else {
		proposal.Status = ProposalCompleted
		if result != nil {
			proposal.Result = result.Message
		}
		m.learningEngine.RecordOutcome(proposal, true, duration)
	}

	if m.store != nil {
		m.store.StoreProposal(proposal)
	}
}

// ManagerStats contains Layer 6 statistics
type ManagerStats struct {
	TotalProposals   int
	PendingProposals int
	TotalIncidents   int
	TotalRules       int
	ByStatus         map[ProposalStatus]int
}
