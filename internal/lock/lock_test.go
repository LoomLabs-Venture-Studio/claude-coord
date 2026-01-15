package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/config"
)

func TestAcquireRelease(t *testing.T) {
	// Setup temp directory
	tmpDir, err := os.MkdirTemp("", "claude-coord-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	coordDir := filepath.Join(tmpDir, ".claude-coord")
	cfg := config.DefaultConfig()
	cfg.Save(coordDir)

	mgr := NewManager(coordDir, cfg)

	// Test acquire
	err = mgr.Acquire("db/schema/*", "agent-1", "Test Agent", "testing", 300)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Test that second acquire fails
	err = mgr.Acquire("db/schema/*", "agent-2", "Other Agent", "also testing", 300)
	if err == nil {
		t.Fatal("Expected error when acquiring already-locked resource")
	}

	// Test release
	err = mgr.Release("db/schema/*", "agent-1")
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Test that acquire now succeeds
	err = mgr.Acquire("db/schema/*", "agent-2", "Other Agent", "also testing", 300)
	if err != nil {
		t.Fatalf("Failed to acquire after release: %v", err)
	}
}

func TestReleaseWrongAgent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude-coord-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	coordDir := filepath.Join(tmpDir, ".claude-coord")
	cfg := config.DefaultConfig()
	cfg.Save(coordDir)

	mgr := NewManager(coordDir, cfg)

	// Acquire as agent-1
	mgr.Acquire("test-resource", "agent-1", "", "", 300)

	// Try to release as agent-2
	err = mgr.Release("test-resource", "agent-2")
	if err == nil {
		t.Fatal("Expected error when wrong agent tries to release")
	}
}

func TestList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude-coord-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	coordDir := filepath.Join(tmpDir, ".claude-coord")
	cfg := config.DefaultConfig()
	cfg.Save(coordDir)

	mgr := NewManager(coordDir, cfg)

	// Create multiple locks
	mgr.Acquire("resource-1", "agent-1", "", "task 1", 300)
	mgr.Acquire("resource-2", "agent-2", "", "task 2", 300)

	locks, err := mgr.List()
	if err != nil {
		t.Fatalf("Failed to list locks: %v", err)
	}

	if len(locks) != 2 {
		t.Fatalf("Expected 2 locks, got %d", len(locks))
	}
}

func TestReleaseAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude-coord-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	coordDir := filepath.Join(tmpDir, ".claude-coord")
	cfg := config.DefaultConfig()
	cfg.Save(coordDir)

	mgr := NewManager(coordDir, cfg)

	// Create locks for same agent
	mgr.Acquire("resource-1", "agent-1", "", "task 1", 300)
	mgr.Acquire("resource-2", "agent-1", "", "task 2", 300)
	mgr.Acquire("resource-3", "agent-2", "", "task 3", 300) // Different agent

	// Release all for agent-1
	err = mgr.ReleaseAll("agent-1")
	if err != nil {
		t.Fatalf("Failed to release all: %v", err)
	}

	locks, _ := mgr.List()
	if len(locks) != 1 {
		t.Fatalf("Expected 1 lock remaining, got %d", len(locks))
	}
	if locks[0].AgentID != "agent-2" {
		t.Fatalf("Wrong lock remaining")
	}
}
