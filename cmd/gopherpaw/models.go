package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suifei/gopherpaw/internal/config"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Model management",
	Long:  "list, set-llm - list or configure LLM models",
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured models",
	RunE:  runModelsList,
}

var modelsSetLLMCmd = &cobra.Command{
	Use:   "set-llm",
	Short: "Set LLM config (simplified: prints config path)",
	RunE:  runModelsSetLLM,
}

func init() {
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsSetLLMCmd)
}

func runModelsList(cmd *cobra.Command, args []string) error {
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
	fmt.Printf("Provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("Model: %s\n", cfg.LLM.Model)
	fmt.Printf("BaseURL: %s\n", cfg.LLM.BaseURL)
	return nil
}

func runModelsSetLLM(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	fmt.Printf("请编辑 %s 中的 llm 配置，或使用 GOPHERPAW_LLM_* 环境变量\n", cfgPath)
	return nil
}
