package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/agent"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var (
	lockOperation string
	lockTTL       int
	lockAgentID   string
	lockAgentName string
)

var lockCmd = &cobra.Command{
	Use:   "lock <resource>",
	Short: "Acquire a lock on a resource",
	Long: `Acquire an exclusive lock on a resource pattern.

The resource should match a pattern from config.yaml, e.g., "db/schema/*".
The lock prevents other agents from modifying files matching this pattern.`,
	Args: cobra.ExactArgs(1),
	RunE: runLock,
}

func init() {
	lockCmd.Flags().StringVar(&lockOperation, "op", "", "Description of what you're doing")
	lockCmd.Flags().IntVar(&lockTTL, "ttl", 0, "Lock timeout in seconds (0 = use default)")
	lockCmd.Flags().StringVar(&lockAgentID, "agent", "", "Agent ID (default: auto-generated)")
	lockCmd.Flags().StringVar(&lockAgentName, "name", "", "Agent display name")
	rootCmd.AddCommand(lockCmd)
}

func runLock(cmd *cobra.Command, args []string) error {
	resource := args[0]

	// Get or generate agent ID
	agentID := lockAgentID
	if agentID == "" {
		agentID = os.Getenv("CLAUDE_SESSION_ID")
		if agentID == "" {
			agentID = agent.GenerateID()
		}
	}

	lockMgr := lock.NewManager(coordDir, cfg)

	if err := lockMgr.Acquire(resource, agentID, lockAgentName, lockOperation, lockTTL); err != nil {
		return err
	}

	fmt.Printf("✓ Locked: %s\n", resource)
	fmt.Printf("  Agent:  %s\n", agentID)
	if lockOperation != "" {
		fmt.Printf("  Task:   %s\n", lockOperation)
	}

	return nil
}
