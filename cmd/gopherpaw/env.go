package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Environment management",
	Long:  "list, set, delete - manage GOPHERPAW_* environment variables",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List GOPHERPAW env vars",
	RunE:  runEnvList,
}

var envSetCmd = &cobra.Command{
	Use:   "set [KEY] [VALUE]",
	Short: "Set env var (prints hint)",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runEnvSet,
}

var envDeleteCmd = &cobra.Command{
	Use:   "delete [KEY]",
	Short: "Delete env var (prints hint)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runEnvDelete,
}

func init() {
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envDeleteCmd)
}

func runEnvList(cmd *cobra.Command, args []string) error {
	prefix := "GOPHERPAW_"
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, prefix) {
			fmt.Println(e)
		}
	}
	return nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	fmt.Println("使用 export GOPHERPAW_<KEY>=<VALUE> 设置环境变量")
	fmt.Println("例如: export GOPHERPAW_LLM_API_KEY=your-key")
	return nil
}

func runEnvDelete(cmd *cobra.Command, args []string) error {
	fmt.Println("使用 unset GOPHERPAW_<KEY> 删除环境变量")
	return nil
}
