package main

import (
	"fmt"
	"os"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/spf13/cobra"
)

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Channel management",
	Long:  "list, config - list or configure channels",
}

var channelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured channels",
	RunE:  runChannelsList,
}

var channelsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show channel config",
	RunE:  runChannelsConfig,
}

func init() {
	channelsCmd.AddCommand(channelsListCmd)
	channelsCmd.AddCommand(channelsConfigCmd)
}

func runChannelsList(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	fmt.Printf("console: enabled=%v\n", cfg.Channels.Console.Enabled)
	fmt.Printf("telegram: enabled=%v\n", cfg.Channels.Telegram.Enabled)
	fmt.Printf("discord: enabled=%v\n", cfg.Channels.Discord.Enabled)
	return nil
}

func runChannelsConfig(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	fmt.Printf("请编辑 %s 中的 channels 配置\n", cfgPath)
	return nil
}
