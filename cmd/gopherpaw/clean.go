package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean working directory",
	Long:  "Remove all files under the working directory (~/.gopherpaw/ by default)",
	RunE:  runClean,
}

func runClean(cmd *cobra.Command, args []string) error {
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
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)
	fmt.Printf("将清空工作目录: %s\n", workingDir)
	fmt.Print("确认? (y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("已取消")
		return nil
	}
	entries, err := os.ReadDir(workingDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("目录不存在，无需清理")
			return nil
		}
		return err
	}
	for _, e := range entries {
		p := filepath.Join(workingDir, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	fmt.Println("已清空")
	return nil
}
