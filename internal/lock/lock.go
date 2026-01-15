package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/config"
)

type Lock struct {
	Resource   string    `json:"resource"`
	AgentID    string    `json:"agent_id"`
	AgentName  string    `json:"agent_name,omitempty"`
	Operation  string    `json:"operation,omitempty"`
	AcquiredAt time.Time `json:"acquired_at"`
	TTLSeconds int       `json:"ttl_seconds"`
	PID        int       `json:"pid"`
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

// Acquire attempts to create a lock for the given resource
func (m *Manager) Acquire(resource, agentID, agentName, operation string, ttl int) error {
	if err := config.EnsureDirs(m.coordDir); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	if ttl == 0 {
		ttl = m.cfg.Settings.DefaultTTL
	}

	lockPath := m.lockPath(resource)

	lock := Lock{
		Resource:   resource,
		AgentID:    agentID,
		AgentName:  agentName,
		Operation:  operation,
		AcquiredAt: time.Now().UTC(),
		TTLSeconds: ttl,
		PID:        os.Getpid(),
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	// O_EXCL ensures atomic creation - fails if file exists
	fd, err := syscall.Open(lockPath, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, 0644)
	if err != nil {
		// Check if existing lock is stale
		existing, readErr := m.Read(resource)
		if readErr == nil {
			if m.IsStale(existing) {
				// Remove stale lock and retry
				if removeErr := os.Remove(lockPath); removeErr == nil {
					return m.Acquire(resource, agentID, agentName, operation, ttl)
				}
			}
			return fmt.Errorf("resource '%s' is locked by agent '%s' (%s): %s",
				resource, existing.AgentID, existing.AgentName, existing.Operation)
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer syscall.Close(fd)

	if _, err := syscall.Write(fd, data); err != nil {
		os.Remove(lockPath)
		return fmt.Errorf("failed to write lock: %w", err)
	}

	return nil
}

// Release removes a lock if owned by the given agent
func (m *Manager) Release(resource, agentID string) error {
	lockPath := m.lockPath(resource)

	existing, err := m.Read(resource)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already unlocked
		}
		return err
	}

	if existing.AgentID != agentID {
		return fmt.Errorf("lock owned by different agent: %s", existing.AgentID)
	}

	return os.Remove(lockPath)
}

// ReleaseAll releases all locks held by the given agent
func (m *Manager) ReleaseAll(agentID string) error {
	locks, err := m.List()
	if err != nil {
		return err
	}

	var errs []error
	for _, lock := range locks {
		if lock.AgentID == agentID {
			if err := m.Release(lock.Resource, agentID); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to release some locks: %v", errs)
	}
	return nil
}

// Read loads a lock from disk
func (m *Manager) Read(resource string) (*Lock, error) {
	lockPath := m.lockPath(resource)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}

	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	return &lock, nil
}

// List returns all current locks
func (m *Manager) List() ([]Lock, error) {
	locksDir := filepath.Join(m.coordDir, config.LocksDir)
	entries, err := os.ReadDir(locksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var locks []Lock
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}

		lockPath := filepath.Join(locksDir, entry.Name())
		data, err := os.ReadFile(lockPath)
		if err != nil {
			continue
		}

		var lock Lock
		if err := json.Unmarshal(data, &lock); err != nil {
			continue
		}
		locks = append(locks, lock)
	}

	return locks, nil
}

// IsStale checks if a lock has expired
func (m *Manager) IsStale(lock *Lock) bool {
	// Check TTL
	if time.Since(lock.AcquiredAt) > time.Duration(lock.TTLSeconds)*time.Second {
		return true
	}

	// Check agent heartbeat
	heartbeatPath := filepath.Join(m.coordDir, config.AgentsDir, lock.AgentID+".agent")
	info, err := os.Stat(heartbeatPath)
	if err != nil {
		// No heartbeat file - check if lock is old enough to be considered stale
		if time.Since(lock.AcquiredAt) > time.Duration(m.cfg.Settings.StaleThreshold)*time.Second {
			return true
		}
		return false
	}

	// Heartbeat exists but is too old
	if time.Since(info.ModTime()) > time.Duration(m.cfg.Settings.StaleThreshold)*time.Second {
		return true
	}

	return false
}

// CleanStale removes all stale locks
func (m *Manager) CleanStale() ([]string, error) {
	locks, err := m.List()
	if err != nil {
		return nil, err
	}

	var cleaned []string
	for _, lock := range locks {
		if m.IsStale(&lock) {
			lockPath := m.lockPath(lock.Resource)
			if err := os.Remove(lockPath); err == nil {
				cleaned = append(cleaned, lock.Resource)
			}
		}
	}

	return cleaned, nil
}

// Check returns the lock if the given file matches a protected pattern and is locked
func (m *Manager) Check(filePath string) (*Lock, bool, error) {
	// First check if file matches any protected pattern
	protected := false
	var matchedPattern string

	for _, p := range m.cfg.Protected {
		matched, err := doublestar.Match(p.Pattern, filePath)
		if err != nil {
			continue
		}
		if matched {
			protected = true
			matchedPattern = p.Pattern
			break
		}
	}

	if !protected {
		return nil, false, nil
	}

	// Check if there's a lock for this pattern
	locks, err := m.List()
	if err != nil {
		return nil, true, err
	}

	for _, lock := range locks {
		// Check if the lock's resource pattern matches
		if lock.Resource == matchedPattern {
			return &lock, true, nil
		}
		// Also check if lock resource matches the file
		matched, _ := doublestar.Match(lock.Resource, filePath)
		if matched {
			return &lock, true, nil
		}
	}

	return nil, true, nil
}

// CheckOrAcquire checks if a file is protected and locked, and acquires if not
func (m *Manager) CheckOrAcquire(filePath, agentID, agentName, operation string) (*Lock, error) {
	lock, protected, err := m.Check(filePath)
	if err != nil {
		return nil, err
	}

	if !protected {
		return nil, nil // Not protected, no lock needed
	}

	if lock != nil {
		if lock.AgentID == agentID {
			return lock, nil // We already have the lock
		}
		return lock, fmt.Errorf("resource locked by %s: %s", lock.AgentID, lock.Operation)
	}

	// Find the matching pattern to use as resource
	var resource string
	for _, p := range m.cfg.Protected {
		matched, _ := doublestar.Match(p.Pattern, filePath)
		if matched {
			resource = p.Pattern
			break
		}
	}

	if err := m.Acquire(resource, agentID, agentName, operation, 0); err != nil {
		return nil, err
	}

	return m.Read(resource)
}

func (m *Manager) lockPath(resource string) string {
	// Convert resource pattern to safe filename
	safe := strings.ReplaceAll(resource, "/", "-")
	safe = strings.ReplaceAll(safe, "\\", "-")
	safe = strings.ReplaceAll(safe, "*", "_")
	safe = strings.ReplaceAll(safe, "?", "_")
	return filepath.Join(m.coordDir, config.LocksDir, safe+".lock")
}
