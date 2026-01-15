package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/config"
)

var (
	initForce      bool
	initRetrofit   bool
	initConfigOnly bool
	initLocal      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize claude-coord in the current directory",
	Long: `Initialize coordination for this repository.

By default, uses .git/claude-coord/ which is shared across all git worktrees.
Use --local to create .claude-coord/ in the current directory instead.

This creates:
  .git/claude-coord/     (or .claude-coord/ with --local)
    config.yaml          - Configuration file (edit this)

And optionally appends coordination instructions to CLAUDE.md.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing configuration")
	initCmd.Flags().BoolVar(&initRetrofit, "retrofit", false, "Set up in existing project (same as default)")
	initCmd.Flags().BoolVar(&initConfigOnly, "config-only", false, "Only create config.yaml, skip CLAUDE.md")
	initCmd.Flags().BoolVar(&initLocal, "local", false, "Use local .claude-coord/ instead of .git/claude-coord/")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	var targetDir string

	if initLocal {
		targetDir = config.DefaultCoordDir
	} else {
		// Try to use git-based directory
		targetDir = config.FindCoordDir()
		if targetDir == config.DefaultCoordDir {
			// Not in a git repo, fall back to local
			fmt.Println("Not in a git repository, using local .claude-coord/")
		}
	}

	// Check if already initialized
	configPath := filepath.Join(targetDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("already initialized at %s (use --force to overwrite)", targetDir)
	}

	// Create directory structure
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create config
	cfg := config.DefaultConfig()
	if err := cfg.Save(targetDir); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("✓ Created %s/config.yaml\n", targetDir)

	// Create .gitignore only for local (non-.git) directories
	isGitBased := strings.Contains(targetDir, ".git")
	if !isGitBased {
		gitignorePath := filepath.Join(targetDir, ".gitignore")
		gitignoreContent := `# Runtime files - don't commit these
locks/
agents/
`
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		fmt.Printf("✓ Created %s/.gitignore\n", targetDir)
	}

	// Create empty runtime directories
	if err := config.EnsureDirs(targetDir); err != nil {
		return fmt.Errorf("failed to create runtime directories: %w", err)
	}

	// Update CLAUDE.md unless --config-only
	if !initConfigOnly {
		if err := updateClaudeMD(); err != nil {
			fmt.Printf("⚠ Could not update CLAUDE.md: %v\n", err)
		} else {
			fmt.Println("✓ Updated CLAUDE.md with coordination instructions")
		}
	}

	fmt.Println("\n✓ Initialized claude-coord")
	
	if isGitBased {
		fmt.Println("\n📁 Using git-based coordination (shared across worktrees)")
		fmt.Printf("   Location: %s\n", targetDir)
	} else {
		fmt.Println("\n📁 Using local coordination")
		fmt.Printf("   Location: %s\n", targetDir)
	}
	
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s/config.yaml to define protected resources\n", targetDir)
	if !isGitBased {
		fmt.Println("  2. Commit .claude-coord/config.yaml and CLAUDE.md")
		fmt.Println("  3. The locks/ and agents/ directories are already gitignored")
	} else {
		fmt.Println("  2. Commit CLAUDE.md to your repository")
		fmt.Println("  3. Coordination files in .git/ are automatically excluded from git")
	}

	return nil
}

func updateClaudeMD() error {
	claudeMDPath := "CLAUDE.md"
	
	content := claudeMDInstructions

	// Check if file exists
	existing, err := os.ReadFile(claudeMDPath)
	if err == nil {
		// File exists - append
		content = string(existing) + "\n\n" + claudeMDInstructions
	}

	return os.WriteFile(claudeMDPath, []byte(content), 0644)
}

const claudeMDInstructions = `## Multi-Agent Coordination

This project uses ` + "`" + `.claude-coord/` + "`" + ` to prevent conflicts when multiple Claude Code agents work simultaneously.

### Quick Reference

` + "```" + `bash
# Check status (always do this first for protected files)
claude-coord status

# Lock a resource before modifying
claude-coord lock "db/schema/*" --op "Adding email verification"

# Release when done
claude-coord unlock "db/schema/*"

# Check if a specific file is protected/locked
claude-coord check path/to/file.sql
` + "```" + `

### Before Modifying Protected Files

1. Check ` + "`" + `.claude-coord/config.yaml` + "`" + ` for protected patterns
2. Run ` + "`" + `claude-coord status` + "`" + ` to see current locks
3. If locked by another agent, **STOP** and tell the user
4. If free, acquire a lock before proceeding
5. Release the lock when finished

### If You See a Lock

Tell the user something like:

> "I need to modify ` + "`" + `db/schema/users.sql` + "`" + `, but another agent is currently working on 'Adding OAuth support' which affects the same files. Would you like me to wait, or should I work on something else first?"

### Protected Resources

See ` + "`" + `.claude-coord/config.yaml` + "`" + ` for the full list. Common patterns:
- ` + "`" + `db/**/*` + "`" + ` - Database schema and migrations
- ` + "`" + `package.json` + "`" + `, ` + "`" + `*.lock` + "`" + ` - Package management
- ` + "`" + `.env*` + "`" + ` - Environment configuration
`
