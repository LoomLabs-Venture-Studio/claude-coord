package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/config"
)

type Agent struct {
	ID            string    `json:"agent_id"`
	Name          string    `json:"name,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CurrentTask   string    `json:"current_task,omitempty"`
	LocksHeld     []string  `json:"locks_held,omitempty"`
	PID           int       `json:"pid"`
}

type Manager struct {
	coordDir string
	cfg      *config.Config
}

func NewManager(coordDir string, cfg *config.Config) *Manager {
	if coordDir == "" {
		coordDir = config.DefaultCoordDir
	}
	return &Manager{
		coordDir: coordDir,
		cfg:      cfg,
	}
}

// Register creates a new agent entry
func (m *Manager) Register(id, name string) error {
	if err := config.EnsureDirs(m.coordDir); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	now := time.Now().UTC()
	agent := Agent{
		ID:            id,
		Name:          name,
		StartedAt:     now,
		LastHeartbeat: now,
		PID:           os.Getpid(),
	}

	return m.save(&agent)
}

// Deregister removes an agent entry
func (m *Manager) Deregister(id string) error {
	agentPath := m.agentPath(id)
	if err := os.Remove(agentPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Heartbeat updates the agent's last heartbeat time
func (m *Manager) Heartbeat(id string) error {
	agent, err := m.Read(id)
	if err != nil {
		// Agent not registered, create minimal entry
		return m.Register(id, "")
	}

	agent.LastHeartbeat = time.Now().UTC()
	return m.save(agent)
}

// UpdateTask updates the agent's current task
func (m *Manager) UpdateTask(id, task string) error {
	agent, err := m.Read(id)
	if err != nil {
		return err
	}

	agent.CurrentTask = task
	agent.LastHeartbeat = time.Now().UTC()
	return m.save(agent)
}

// UpdateLocks updates the agent's held locks
func (m *Manager) UpdateLocks(id string, locks []string) error {
	agent, err := m.Read(id)
	if err != nil {
		return err
	}

	agent.LocksHeld = locks
	agent.LastHeartbeat = time.Now().UTC()
	return m.save(agent)
}

// Read loads an agent from disk
func (m *Manager) Read(id string) (*Agent, error) {
	agentPath := m.agentPath(id)
	data, err := os.ReadFile(agentPath)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

// List returns all registered agents
func (m *Manager) List() ([]Agent, error) {
	agentsDir := filepath.Join(m.coordDir, config.AgentsDir)
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []Agent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".agent") {
			continue
		}

		agentPath := filepath.Join(agentsDir, entry.Name())
		data, err := os.ReadFile(agentPath)
		if err != nil {
			continue
		}

		var agent Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// IsAlive checks if an agent is still alive based on heartbeat
func (m *Manager) IsAlive(agent *Agent) bool {
	threshold := time.Duration(m.cfg.Settings.StaleThreshold) * time.Second
	return time.Since(agent.LastHeartbeat) < threshold
}

// CleanStale removes dead agent entries
func (m *Manager) CleanStale() ([]string, error) {
	agents, err := m.List()
	if err != nil {
		return nil, err
	}

	var cleaned []string
	for _, agent := range agents {
		if !m.IsAlive(&agent) {
			if err := m.Deregister(agent.ID); err == nil {
				cleaned = append(cleaned, agent.ID)
			}
		}
	}

	return cleaned, nil
}

// RunHeartbeat runs a heartbeat loop in the background
func (m *Manager) RunHeartbeat(id string, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.Heartbeat(id)
		case <-stop:
			return
		}
	}
}

func (m *Manager) save(agent *Agent) error {
	data, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return err
	}

	agentPath := m.agentPath(agent.ID)
	return os.WriteFile(agentPath, data, 0644)
}

func (m *Manager) agentPath(id string) string {
	// Sanitize ID for filename
	safe := strings.ReplaceAll(id, "/", "-")
	safe = strings.ReplaceAll(safe, "\\", "-")
	return filepath.Join(m.coordDir, config.AgentsDir, safe+".agent")
}

// GenerateID creates a unique agent ID
func GenerateID() string {
	return fmt.Sprintf("agent-%d-%d", os.Getpid(), time.Now().UnixNano()%100000)
}
