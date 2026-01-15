package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/agent"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current locks and agents",
	Long:  `Display all active locks and registered agents with their current status.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	lockMgr := lock.NewManager(coordDir, cfg)
	agentMgr := agent.NewManager(coordDir, cfg)

	// Get locks
	locks, err := lockMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list locks: %w", err)
	}

	// Get agents
	agents, err := agentMgr.List()
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	// Display locks
	fmt.Println("LOCKS")
	fmt.Println("─────")
	if len(locks) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, l := range locks {
			stale := ""
			if lockMgr.IsStale(&l) {
				stale = " [STALE]"
			}
			age := time.Since(l.AcquiredAt).Round(time.Second)
			fmt.Printf("  • %s%s\n", l.Resource, stale)
			fmt.Printf("    Agent: %s", l.AgentID)
			if l.AgentName != "" {
				fmt.Printf(" (%s)", l.AgentName)
			}
			fmt.Println()
			if l.Operation != "" {
				fmt.Printf("    Task:  %s\n", l.Operation)
			}
			fmt.Printf("    Age:   %s (TTL: %ds)\n", age, l.TTLSeconds)
		}
	}

	fmt.Println()

	// Display agents
	fmt.Println("AGENTS")
	fmt.Println("──────")
	if len(agents) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, a := range agents {
			status := "alive"
			if !agentMgr.IsAlive(&a) {
				status = "dead"
			}
			lastSeen := time.Since(a.LastHeartbeat).Round(time.Second)
			fmt.Printf("  • %s", a.ID)
			if a.Name != "" {
				fmt.Printf(" (%s)", a.Name)
			}
			fmt.Printf(" [%s]\n", status)
			fmt.Printf("    Last seen: %s ago\n", lastSeen)
			if a.CurrentTask != "" {
				fmt.Printf("    Task: %s\n", a.CurrentTask)
			}
			if len(a.LocksHeld) > 0 {
				fmt.Printf("    Locks: %v\n", a.LocksHeld)
			}
		}
	}

	return nil
}
