package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/agent"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Clean up stale locks and dead agents",
	Long: `Remove locks that have exceeded their TTL and agents that 
haven't sent a heartbeat within the stale threshold.`,
	RunE: runGC,
}

func init() {
	rootCmd.AddCommand(gcCmd)
}

func runGC(cmd *cobra.Command, args []string) error {
	lockMgr := lock.NewManager(coordDir, cfg)
	agentMgr := agent.NewManager(coordDir, cfg)

	// Clean stale locks
	cleanedLocks, err := lockMgr.CleanStale()
	if err != nil {
		return fmt.Errorf("failed to clean locks: %w", err)
	}

	// Clean dead agents
	cleanedAgents, err := agentMgr.CleanStale()
	if err != nil {
		return fmt.Errorf("failed to clean agents: %w", err)
	}

	if len(cleanedLocks) == 0 && len(cleanedAgents) == 0 {
		fmt.Println("✓ Nothing to clean")
		return nil
	}

	if len(cleanedLocks) > 0 {
		fmt.Printf("✓ Cleaned %d stale lock(s):\n", len(cleanedLocks))
		for _, l := range cleanedLocks {
			fmt.Printf("  • %s\n", l)
		}
	}

	if len(cleanedAgents) > 0 {
		fmt.Printf("✓ Cleaned %d dead agent(s):\n", len(cleanedAgents))
		for _, a := range cleanedAgents {
			fmt.Printf("  • %s\n", a)
		}
	}

	return nil
}
