package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/agent"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/cache"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var (
	checkAcquire   bool
	checkAgentID   string
	checkAgentName string
	checkOperation string
)

var checkCmd = &cobra.Command{
	Use:   "check <file> [file...]",
	Short: "Check if files are protected and/or locked",
	Long: `Check if one or more files match a protected pattern and if they're currently locked.

With --acquire, automatically acquire locks for protected files that aren't locked.
Exit code is non-zero if any file is locked by another agent.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkAcquire, "acquire", false, "Auto-acquire locks for unlocked protected files")
	checkCmd.Flags().StringVar(&checkAgentID, "agent", "", "Agent ID for acquiring locks")
	checkCmd.Flags().StringVar(&checkAgentName, "name", "", "Agent display name")
	checkCmd.Flags().StringVar(&checkOperation, "op", "", "Operation description for acquired locks")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Get agent ID
	agentID := checkAgentID
	if agentID == "" {
		agentID = os.Getenv("CLAUDE_SESSION_ID")
		if agentID == "" {
			agentID = agent.GenerateID()
		}
	}

	lockMgr := lock.NewManager(coordDir, cfg)

	// Load cache for fast "not protected" lookups
	checkCache := cache.Load(coordDir)
	if !checkCache.IsValid(cfg) {
		checkCache.Reset(cfg)
	}
	cacheModified := false

	var blocked []string
	var acquired []string

	for _, file := range args {
		// Handle space/comma separated file lists (from hooks)
		files := splitFiles(file)
		for _, f := range files {
			// Fast path: check cache first
			if checkCache.IsNotProtected(f) {
				continue // Skip - we know this file isn't protected
			}

			if checkAcquire {
				existingLock, err := lockMgr.CheckOrAcquire(f, agentID, checkAgentName, checkOperation)
				if err != nil {
					// Blocked by another agent
					blocked = append(blocked, fmt.Sprintf("%s (locked by %s: %s)",
						f, existingLock.AgentID, existingLock.Operation))
				} else if existingLock != nil {
					acquired = append(acquired, f)
				} else {
					// Not protected - cache it
					checkCache.MarkNotProtected(f)
					cacheModified = true
				}
			} else {
				existingLock, protected, err := lockMgr.Check(f)
				if err != nil {
					return err
				}
				if !protected {
					// Not protected - cache it
					checkCache.MarkNotProtected(f)
					cacheModified = true
				} else if existingLock != nil && existingLock.AgentID != agentID {
					blocked = append(blocked, fmt.Sprintf("%s (locked by %s: %s)",
						f, existingLock.AgentID, existingLock.Operation))
				}
			}
		}
	}

	// Save cache if modified
	if cacheModified {
		checkCache.Save()
	}

	// Only output when something notable happens
	if len(acquired) > 0 {
		fmt.Printf("✓ Acquired locks for: %s\n", strings.Join(acquired, ", "))
	}

	if len(blocked) > 0 {
		fmt.Println("✗ Blocked:")
		for _, b := range blocked {
			fmt.Printf("  • %s\n", b)
		}
		return fmt.Errorf("blocked by %d lock(s)", len(blocked))
	}

	return nil
}

func splitFiles(input string) []string {
	// Handle various separators that might come from hooks
	input = strings.ReplaceAll(input, ",", " ")
	input = strings.ReplaceAll(input, ";", " ")
	
	var files []string
	for _, f := range strings.Fields(input) {
		f = strings.TrimSpace(f)
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}
