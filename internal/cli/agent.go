package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/agent"
	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/lock"
)

var (
	registerAgentID   string
	registerAgentName string
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register this agent",
	Long:  `Register this Claude agent with the coordination system.`,
	RunE:  runRegister,
}

var (
	heartbeatAgentID  string
	heartbeatDaemon   bool
	heartbeatInterval int
)

var heartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "Send a heartbeat for this agent",
	Long: `Update the agent's heartbeat timestamp.

With --daemon, runs continuously in the background until killed.`,
	RunE: runHeartbeat,
}

var (
	deregisterAgentID   string
	deregisterRelease   bool
)

var deregisterCmd = &cobra.Command{
	Use:   "deregister",
	Short: "Deregister this agent",
	Long:  `Remove this agent from the coordination system.`,
	RunE:  runDeregister,
}

func init() {
	registerCmd.Flags().StringVar(&registerAgentID, "agent", "", "Agent ID (default: from env or auto)")
	registerCmd.Flags().StringVar(&registerAgentName, "name", "", "Agent display name")
	rootCmd.AddCommand(registerCmd)

	heartbeatCmd.Flags().StringVar(&heartbeatAgentID, "agent", "", "Agent ID")
	heartbeatCmd.Flags().BoolVar(&heartbeatDaemon, "daemon", false, "Run continuously")
	heartbeatCmd.Flags().IntVar(&heartbeatInterval, "interval", 0, "Heartbeat interval in seconds (0 = use config)")
	rootCmd.AddCommand(heartbeatCmd)

	deregisterCmd.Flags().StringVar(&deregisterAgentID, "agent", "", "Agent ID")
	deregisterCmd.Flags().BoolVar(&deregisterRelease, "release-all", false, "Release all locks held by this agent")
	rootCmd.AddCommand(deregisterCmd)
}

func runRegister(cmd *cobra.Command, args []string) error {
	agentID := registerAgentID
	if agentID == "" {
		agentID = os.Getenv("CLAUDE_SESSION_ID")
		if agentID == "" {
			agentID = agent.GenerateID()
		}
	}

	agentMgr := agent.NewManager(coordDir, cfg)

	if err := agentMgr.Register(agentID, registerAgentName); err != nil {
		return err
	}

	fmt.Printf("✓ Registered agent: %s\n", agentID)
	if registerAgentName != "" {
		fmt.Printf("  Name: %s\n", registerAgentName)
	}

	return nil
}

func runHeartbeat(cmd *cobra.Command, args []string) error {
	agentID := heartbeatAgentID
	if agentID == "" {
		agentID = os.Getenv("CLAUDE_SESSION_ID")
		if agentID == "" {
			agentID = agent.GenerateID()
		}
	}

	agentMgr := agent.NewManager(coordDir, cfg)

	if !heartbeatDaemon {
		// Single heartbeat
		if err := agentMgr.Heartbeat(agentID); err != nil {
			return err
		}
		fmt.Printf("✓ Heartbeat sent for: %s\n", agentID)
		return nil
	}

	// Daemon mode
	interval := heartbeatInterval
	if interval == 0 {
		interval = cfg.Settings.HeartbeatInterval
	}

	fmt.Printf("Starting heartbeat daemon for %s (interval: %ds)\n", agentID, interval)

	stop := make(chan struct{})
	
	// Handle signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		close(stop)
	}()

	agentMgr.RunHeartbeat(agentID, time.Duration(interval)*time.Second, stop)
	fmt.Println("Heartbeat daemon stopped")

	return nil
}

func runDeregister(cmd *cobra.Command, args []string) error {
	agentID := deregisterAgentID
	if agentID == "" {
		agentID = os.Getenv("CLAUDE_SESSION_ID")
		if agentID == "" {
			return fmt.Errorf("agent ID required")
		}
	}

	// Release locks if requested
	if deregisterRelease {
		lockMgr := lock.NewManager(coordDir, cfg)
		if err := lockMgr.ReleaseAll(agentID); err != nil {
			fmt.Printf("⚠ Warning: failed to release some locks: %v\n", err)
		} else {
			fmt.Printf("✓ Released all locks for: %s\n", agentID)
		}
	}

	agentMgr := agent.NewManager(coordDir, cfg)
	if err := agentMgr.Deregister(agentID); err != nil {
		return err
	}

	fmt.Printf("✓ Deregistered agent: %s\n", agentID)
	return nil
}
