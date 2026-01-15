package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultCoordDir    = ".claude-coord"
	ConfigFileName     = "config.yaml"
	LocksDir           = "locks"
	AgentsDir          = "agents"
	DefaultTTL         = 300
	DefaultStale       = 120
	DefaultHeartbeat   = 30
)

type Config struct {
	Version   int              `yaml:"version"`
	Protected []ProtectedPath  `yaml:"protected"`
	Logical   []LogicalResource `yaml:"logical,omitempty"`
	Settings  Settings         `yaml:"settings"`
}

type ProtectedPath struct {
	Pattern     string `yaml:"pattern"`
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type LogicalResource struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Files       []string `yaml:"files,omitempty"`
}

type Settings struct {
	DefaultTTL        int `yaml:"default_ttl"`
	StaleThreshold    int `yaml:"stale_threshold"`
	HeartbeatInterval int `yaml:"heartbeat_interval"`
}

// Load reads the config from the given directory
func Load(coordDir string) (*Config, error) {
	if coordDir == "" {
		coordDir = DefaultCoordDir
	}

	configPath := filepath.Join(coordDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.Settings.DefaultTTL == 0 {
		cfg.Settings.DefaultTTL = DefaultTTL
	}
	if cfg.Settings.StaleThreshold == 0 {
		cfg.Settings.StaleThreshold = DefaultStale
	}
	if cfg.Settings.HeartbeatInterval == 0 {
		cfg.Settings.HeartbeatInterval = DefaultHeartbeat
	}

	return &cfg, nil
}

// Save writes the config to the given directory
func (c *Config) Save(coordDir string) error {
	if coordDir == "" {
		coordDir = DefaultCoordDir
	}

	if err := os.MkdirAll(coordDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	configPath := filepath.Join(coordDir, ConfigFileName)
	return os.WriteFile(configPath, data, 0644)
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Protected: []ProtectedPath{
			{Pattern: "db/**/*", Name: "Database", Description: "Database schema and migrations"},
			{Pattern: "migrations/**/*", Name: "Migrations"},
			{Pattern: "prisma/schema.prisma", Name: "Prisma Schema"},
			{Pattern: "drizzle/**/*", Name: "Drizzle Schema"},
			{Pattern: "package.json", Name: "NPM Config"},
			{Pattern: "package-lock.json", Name: "NPM Lock"},
			{Pattern: "yarn.lock", Name: "Yarn Lock"},
			{Pattern: "pnpm-lock.yaml", Name: "PNPM Lock"},
			{Pattern: "Cargo.toml", Name: "Cargo Config"},
			{Pattern: "Cargo.lock", Name: "Cargo Lock"},
			{Pattern: "go.mod", Name: "Go Module"},
			{Pattern: "go.sum", Name: "Go Sum"},
			{Pattern: "requirements.txt", Name: "Python Requirements"},
			{Pattern: "pyproject.toml", Name: "Python Project"},
			{Pattern: "poetry.lock", Name: "Poetry Lock"},
			{Pattern: ".env*", Name: "Environment Files"},
		},
		Settings: Settings{
			DefaultTTL:        DefaultTTL,
			StaleThreshold:    DefaultStale,
			HeartbeatInterval: DefaultHeartbeat,
		},
	}
}

// EnsureDirs creates the locks and agents directories
func EnsureDirs(coordDir string) error {
	if coordDir == "" {
		coordDir = DefaultCoordDir
	}

	dirs := []string{
		filepath.Join(coordDir, LocksDir),
		filepath.Join(coordDir, AgentsDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// FindCoordDir searches for the coordination directory.
// It first checks for a git repo and uses .git/claude-coord/ for worktree support.
// Falls back to local .claude-coord/ if not in a git repo.
func FindCoordDir() string {
	// First, try to use git's shared directory (supports worktrees)
	if gitDir := findGitCoordDir(); gitDir != "" {
		return gitDir
	}

	// Fallback: search for local .claude-coord/
	return findLocalCoordDir()
}

// findGitCoordDir returns the coordination directory inside .git/
// This is shared across all worktrees of the same repository.
func findGitCoordDir() string {
	// Get the common git directory (shared across worktrees)
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	commonDir := strings.TrimSpace(string(output))
	if commonDir == "" || commonDir == ".git" {
		// Resolve relative path
		cwd, err := os.Getwd()
		if err != nil {
			return ""
		}
		commonDir = filepath.Join(cwd, ".git")
	}

	// Use .git/claude-coord/ as the shared coordination directory
	coordDir := filepath.Join(commonDir, "claude-coord")
	
	// Check if it exists or if config exists in it
	configPath := filepath.Join(coordDir, ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return coordDir
	}

	// Also check if locks dir exists (might be initialized but no config yet)
	locksDir := filepath.Join(coordDir, LocksDir)
	if _, err := os.Stat(locksDir); err == nil {
		return coordDir
	}

	// Return the git-based path anyway if we're in a git repo
	// init command will create it
	if _, err := os.Stat(commonDir); err == nil {
		return coordDir
	}

	return ""
}

// findLocalCoordDir searches for .claude-coord in current and parent directories
func findLocalCoordDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return DefaultCoordDir
	}

	for {
		candidate := filepath.Join(dir, DefaultCoordDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return DefaultCoordDir
}
