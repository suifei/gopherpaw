package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Cron job management",
	Long:  "list, create, delete, pause, resume - manage cron jobs",
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cron jobs",
	RunE:  runCronList,
}

var cronCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create cron job (prints hint)",
	RunE:  runCronCreate,
}

var cronDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete cron job (prints hint)",
	RunE:  runCronDelete,
}

var cronPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause cron job (prints hint)",
	RunE:  runCronPause,
}

var cronResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume cron job (prints hint)",
	RunE:  runCronResume,
}

func init() {
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronCreateCmd)
	cronCmd.AddCommand(cronDeleteCmd)
	cronCmd.AddCommand(cronPauseCmd)
	cronCmd.AddCommand(cronResumeCmd)
}

func runCronList(cmd *cobra.Command, args []string) error {
	fmt.Println("Cron 任务需在 app 运行时通过 scheduler 管理")
	fmt.Println("请在 configs/config.yaml 中配置 scheduler.enabled")
	return nil
}

func runCronCreate(cmd *cobra.Command, args []string) error {
	fmt.Println("在 configs/config.yaml 的 scheduler 中配置 cron 表达式")
	return nil
}

func runCronDelete(cmd *cobra.Command, args []string) error {
	fmt.Println("编辑 configs/config.yaml 移除对应 cron 任务")
	return nil
}

func runCronPause(cmd *cobra.Command, args []string) error {
	fmt.Println("设置 scheduler.enabled: false 暂停所有任务")
	return nil
}

func runCronResume(cmd *cobra.Command, args []string) error {
	fmt.Println("设置 scheduler.enabled: true 恢复任务")
	return nil
}
