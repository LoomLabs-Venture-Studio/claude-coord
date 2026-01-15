package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var (
	waitTimeout  int
	waitInterval int
)

var waitCmd = &cobra.Command{
	Use:   "wait <resource>",
	Short: "Wait for a resource to become available",
	Long: `Block until the specified resource is no longer locked.

Useful for coordinating sequential tasks between agents.`,
	Args: cobra.ExactArgs(1),
	RunE: runWait,
}

func init() {
	waitCmd.Flags().IntVar(&waitTimeout, "timeout", 300, "Maximum time to wait in seconds (0 = infinite)")
	waitCmd.Flags().IntVar(&waitInterval, "interval", 5, "Check interval in seconds")
	rootCmd.AddCommand(waitCmd)
}

func runWait(cmd *cobra.Command, args []string) error {
	resource := args[0]
	lockMgr := lock.NewManager(coordDir, cfg)

	start := time.Now()
	checkInterval := time.Duration(waitInterval) * time.Second

	fmt.Printf("Waiting for %s to become available...\n", resource)

	for {
		// Check if lock exists
		existingLock, err := lockMgr.Read(resource)
		if err != nil {
			// No lock - resource is free
			fmt.Printf("✓ Resource available: %s\n", resource)
			return nil
		}

		// Check if stale
		if lockMgr.IsStale(existingLock) {
			fmt.Printf("✓ Lock was stale, resource available: %s\n", resource)
			return nil
		}

		// Check timeout
		if waitTimeout > 0 && time.Since(start) > time.Duration(waitTimeout)*time.Second {
			return fmt.Errorf("timeout waiting for %s (locked by %s)", resource, existingLock.AgentID)
		}

		elapsed := time.Since(start).Round(time.Second)
		fmt.Printf("  Still locked by %s (%s), waiting... (%s elapsed)\n",
			existingLock.AgentID, existingLock.Operation, elapsed)

		time.Sleep(checkInterval)
	}
}
