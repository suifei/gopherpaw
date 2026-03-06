package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive initialization",
	Long:  "Interactively configure LLM, channels, heartbeat, language, and skills. Run 'gopherpaw init' to start.",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("GopherPaw 交互式初始化")
	fmt.Println("请编辑 configs/config.yaml 配置以下项：")
	fmt.Println("  - llm: provider, model, api_key, base_url")
	fmt.Println("  - channels: console/telegram/discord 等")
	fmt.Println("  - scheduler: 定时任务")
	fmt.Println("  - skills: active_dir, customized_dir")
	fmt.Println("")
	fmt.Println("或使用环境变量 GOPHERPAW_* 覆盖，如 GOPHERPAW_LLM_API_KEY")
	return nil
}
