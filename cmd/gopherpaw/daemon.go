package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon management",
	Long:  "status, version, logs - daemon management commands",
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	RunE:  runDaemonStatus,
}

var daemonVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	RunE:  runDaemonVersion,
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs [N]",
	Short: "Show last N lines of logs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDaemonLogs,
}

func init() {
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonVersionCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("status: standalone (no daemon mode)")
	return nil
}

func runDaemonVersion(cmd *cobra.Command, args []string) error {
	fmt.Println("gopherpaw", version)
	return nil
}

func runDaemonLogs(cmd *cobra.Command, args []string) error {
	n := 20
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil || n <= 0 {
			n = 20
		}
	}
	fmt.Printf("最近 %d 行日志: (日志功能需在 app 运行时通过 /daemon logs 获取)\n", n)
	return nil
}
