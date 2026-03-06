package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var chatsCmd = &cobra.Command{
	Use:   "chats",
	Short: "Chat management",
	Long:  "list, get, delete - manage chat sessions",
}

var chatsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List chats",
	RunE:  runChatsList,
}

var chatsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get chat (prints hint)",
	RunE:  runChatsGet,
}

var chatsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete chat (prints hint)",
	RunE:  runChatsDelete,
}

func init() {
	chatsCmd.AddCommand(chatsListCmd)
	chatsCmd.AddCommand(chatsGetCmd)
	chatsCmd.AddCommand(chatsDeleteCmd)
}

func runChatsList(cmd *cobra.Command, args []string) error {
	fmt.Println("Chat 会话存储在 memory 中，当前为内存模式")
	fmt.Println("使用 /history 查看当前对话历史")
	return nil
}

func runChatsGet(cmd *cobra.Command, args []string) error {
	fmt.Println("在 Console 渠道中，chatID 为 console:default")
	return nil
}

func runChatsDelete(cmd *cobra.Command, args []string) error {
	fmt.Println("使用 /clear 清空当前对话上下文")
	return nil
}
