// Package main provides the GopherPaw CLI entry point.
package main

import (
	"github.com/spf13/cobra"
)

// rootCmd is the root command. Default action: run app (gopherpaw = gopherpaw app).
var rootCmd = &cobra.Command{
	Use:   "gopherpaw",
	Short: "GopherPaw - CoPaw Go language reimplementation",
	Long:  "GopherPaw is the CoPaw AI Agent reimplementation in Go. Use subcommands to manage config, channels, and run the service.",
	RunE:  runApp,
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path (default: configs/config.yaml)")
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(channelsCmd)
	rootCmd.AddCommand(cronCmd)
	rootCmd.AddCommand(chatsCmd)
	rootCmd.AddCommand(skillsCmd)
	rootCmd.AddCommand(cleanCmd)
}
