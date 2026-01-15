package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/agent"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var (
	unlockAll     bool
	unlockAgentID string
)

var unlockCmd = &cobra.Command{
	Use:   "unlock [resource]",
	Short: "Release a lock on a resource",
	Long: `Release a lock that you previously acquired.

Use --all to release all locks held by your agent.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUnlock,
}

func init() {
	unlockCmd.Flags().BoolVar(&unlockAll, "all", false, "Release all locks held by this agent")
	unlockCmd.Flags().StringVar(&unlockAgentID, "agent", "", "Agent ID (default: from env or auto)")
	rootCmd.AddCommand(unlockCmd)
}

func runUnlock(cmd *cobra.Command, args []string) error {
	// Get agent ID
	agentID := unlockAgentID
	if agentID == "" {
		agentID = os.Getenv("CLAUDE_SESSION_ID")
		if agentID == "" {
			agentID = agent.GenerateID()
		}
	}

	lockMgr := lock.NewManager(coordDir, cfg)

	if unlockAll {
		if err := lockMgr.ReleaseAll(agentID); err != nil {
			return err
		}
		fmt.Printf("✓ Released all locks for agent: %s\n", agentID)
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a resource or use --all")
	}

	resource := args[0]
	if err := lockMgr.Release(resource, agentID); err != nil {
		return err
	}

	fmt.Printf("✓ Released: %s\n", resource)
	return nil
}
