package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/config"
)

const CacheFileName = "check-cache.json"

type Cache struct {
	ConfigHash   string            `json:"config_hash"`
	NotProtected map[string]bool   `json:"not_protected"` // files we know aren't protected
	mu           sync.RWMutex
	coordDir     string
}

// Load reads the cache from disk, or returns a new empty cache
func Load(coordDir string) *Cache {
	c := &Cache{
		NotProtected: make(map[string]bool),
		coordDir:     coordDir,
	}

	cachePath := filepath.Join(coordDir, CacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return c
	}

	json.Unmarshal(data, c)
	c.coordDir = coordDir
	return c
}

// IsValid checks if the cache is still valid for the given config
func (c *Cache) IsValid(cfg *config.Config) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ConfigHash == hashConfig(cfg)
}

// Reset clears the cache and sets the new config hash
func (c *Cache) Reset(cfg *config.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ConfigHash = hashConfig(cfg)
	c.NotProtected = make(map[string]bool)
}

// IsNotProtected returns true if we've cached that this file is not protected
func (c *Cache) IsNotProtected(filePath string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.NotProtected[filePath]
}

// MarkNotProtected marks a file as not protected in the cache
func (c *Cache) MarkNotProtected(filePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.NotProtected[filePath] = true
}

// Save writes the cache to disk
func (c *Cache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(c.coordDir, CacheFileName)
	return os.WriteFile(cachePath, data, 0644)
}

func hashConfig(cfg *config.Config) string {
	// Hash just the protected patterns since that's what affects check results
	var patterns []string
	for _, p := range cfg.Protected {
		patterns = append(patterns, p.Pattern)
	}

	data, _ := json.Marshal(patterns)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8]) // short hash is fine
}
