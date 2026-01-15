package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/config"
)

var (
	coordDir string
	cfg      *config.Config
	version  = "dev"
)

// SetVersion sets the version string (called from main)
func SetVersion(v string) {
	version = v
}

var rootCmd = &cobra.Command{
	Use:   "claude-coord",
	Short: "Coordinate multiple Claude Code agents",
	Long: `claude-coord provides a lightweight coordination system for multiple 
Claude Code agents working in the same codebase.

It prevents conflicts by allowing agents to lock resources before 
modifying them, and checking for existing locks before proceeding.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for init command
		if cmd.Name() == "init" {
			return nil
		}

		// Find and load config
		if coordDir == "" {
			coordDir = config.FindCoordDir()
		}

		var err error
		cfg, err = config.Load(coordDir)
		if err != nil {
			if os.IsNotExist(err) {
				// Config doesn't exist - some commands can work without it
				cfg = config.DefaultConfig()
				return nil
			}
			return fmt.Errorf("failed to load config: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&coordDir, "dir", "", "Path to .claude-coord directory")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("claude-coord %s\n", version)
	},
}

func Execute() error {
	return rootCmd.Execute()
}
