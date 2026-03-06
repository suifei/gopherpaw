// Package channels provides message channel adapters.
package channels

import "context"

// Channel represents a messaging platform adapter.
type Channel interface {
	// Start begins listening for messages.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the channel.
	Stop(ctx context.Context) error

	// Name returns the channel identifier.
	Name() string

	// Send sends a text message to the given target.
	Send(ctx context.Context, to string, text string, meta map[string]string) error

	// IsEnabled returns true if the channel is enabled in config.
	IsEnabled() bool
}

// FileSender is an optional Channel extension for delivering file attachments.
type FileSender interface {
	SendFile(ctx context.Context, to string, filePath string, mime string, meta map[string]string) error
}

// ImageSender is an optional Channel extension for sending images directly.
type ImageSender interface {
	SendImage(ctx context.Context, to string, imageData []byte, filename string, meta map[string]string) error
}

// MarkdownSender is an optional Channel extension for sending Markdown-formatted messages.
// Platforms that support rich formatting (Telegram, Discord, Feishu) can implement this.
type MarkdownSender interface {
	SendMarkdown(ctx context.Context, to string, markdown string, meta map[string]string) error
}

// MessageEditor is an optional Channel extension for editing previously sent messages.
// The messageID is typically returned from Send/SendMarkdown via meta["message_id"].
type MessageEditor interface {
	EditMessage(ctx context.Context, to string, messageID string, newText string, meta map[string]string) error
}

// MessageDeleter is an optional Channel extension for deleting messages.
type MessageDeleter interface {
	DeleteMessage(ctx context.Context, to string, messageID string, meta map[string]string) error
}

// Reactor is an optional Channel extension for adding reactions to messages.
type Reactor interface {
	React(ctx context.Context, to string, messageID string, emoji string, meta map[string]string) error
}

// TypingIndicator is an optional Channel extension for showing typing status.
type TypingIndicator interface {
	SendTyping(ctx context.Context, to string, meta map[string]string) error
}

// IncomingMessage represents a message received from a channel.
type IncomingMessage struct {
	ChatID    string            `json:"chat_id"`
	UserID    string            `json:"user_id"`
	UserName  string            `json:"user_name"`
	Content   string            `json:"content"`
	Channel   string            `json:"channel"`
	Timestamp int64             `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
