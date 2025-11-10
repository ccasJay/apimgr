package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"apimgr/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Manage apimgr daemon",
	Hidden: true, // Hide from main help
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the apimgr daemon",
	Run:   runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the apimgr daemon",
	Run:   runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Run:   runDaemonStatus,
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the apimgr daemon",
	Run:   runDaemonRestart,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
}

func getDirectories() (string, string) {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "apimgr")
	
	uid := os.Getuid()
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	runtimeDir = filepath.Join(runtimeDir, fmt.Sprintf("apimgr-%d", uid))
	
	return configDir, runtimeDir
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	configDir, runtimeDir := getDirectories()
	
	// Create daemon instance
	d, err := daemon.New(configDir, runtimeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create daemon: %v\n", err)
		os.Exit(1)
	}
	
	// Check if already running
	if d.IsRunning() {
		fmt.Println("Daemon is already running")
		return
	}
	
	// Fork to background if not already in background
	if os.Getppid() != 1 {
		// We're not already daemonized, so fork
		executable, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to get executable path: %v\n", err)
			os.Exit(1)
		}
		
		// Start new process in background
		procAttr := &os.ProcAttr{
			Dir:   "/",
			Env:   os.Environ(),
			Files: []*os.File{nil, nil, nil}, // Detach from terminal
		}
		
		process, err := os.StartProcess(executable, []string{executable, "daemon", "start"}, procAttr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to start daemon: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Daemon started (PID: %d)\n", process.Pid)
		process.Release()
		return
	}
	
	// We're in the background process, start the actual daemon
	if err := d.Start(); err != nil {
		// Silent exit in background
		os.Exit(1)
	}
	
	// Keep daemon running
	select {}
}

func runDaemonStop(cmd *cobra.Command, args []string) {
	_, runtimeDir := getDirectories()
	
	pidPath := filepath.Join(runtimeDir, "daemon.pid")
	if data, err := os.ReadFile(pidPath); err == nil {
		var pid int
		if _, err := fmt.Sscanf(string(data), "%d", &pid); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				if err := process.Signal(syscall.SIGTERM); err == nil {
					fmt.Printf("Daemon stopped (PID: %d)\n", pid)
					return
				}
			}
		}
	}
	
	fmt.Println("Daemon is not running")
}

func runDaemonStatus(cmd *cobra.Command, args []string) {
	configDir, runtimeDir := getDirectories()
	
	d, err := daemon.New(configDir, runtimeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create daemon: %v\n", err)
		os.Exit(1)
	}
	
	if d.IsRunning() {
		pidPath := filepath.Join(runtimeDir, "daemon.pid")
		if data, err := os.ReadFile(pidPath); err == nil {
			fmt.Printf("Daemon is running (PID: %s)\n", string(data))
		} else {
			fmt.Println("Daemon is running")
		}
		
		socketPath := filepath.Join(runtimeDir, "apimgr.sock")
		if _, err := os.Stat(socketPath); err == nil {
			fmt.Printf("Socket: %s\n", socketPath)
		}
	} else {
		fmt.Println("Daemon is not running")
	}
}

func runDaemonRestart(cmd *cobra.Command, args []string) {
	runDaemonStop(cmd, args)
	runDaemonStart(cmd, args)
}
