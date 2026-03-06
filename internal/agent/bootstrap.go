// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// BootstrapRunner runs the first-time bootstrap flow when BOOTSTRAP.md exists.
type BootstrapRunner struct {
	agent  Agent
	loader *PromptLoader
}

// NewBootstrapRunner creates a BootstrapRunner.
func NewBootstrapRunner(agent Agent, loader *PromptLoader) *BootstrapRunner {
	return &BootstrapRunner{
		agent:  agent,
		loader: loader,
	}
}

// RunIfNeeded checks for BOOTSTRAP.md and runs the bootstrap flow if present.
// Flow: read BOOTSTRAP.md → Agent.Run(chatID, content) → write response to PROFILE.md → delete BOOTSTRAP.md.
func (b *BootstrapRunner) RunIfNeeded(ctx context.Context, chatID string) error {
	if !b.loader.HasBootstrap() {
		return nil
	}
	log := logger.L()
	log.Info("Bootstrap detected, running first-time setup")

	bootstrapContent, err := b.readBootstrap()
	if err != nil {
		return fmt.Errorf("read BOOTSTRAP: %w", err)
	}
	if bootstrapContent == "" {
		_ = b.loader.DeleteBootstrap()
		return nil
	}

	response, err := b.agent.Run(ctx, chatID, bootstrapContent)
	if err != nil {
		return fmt.Errorf("bootstrap agent run: %w", err)
	}

	profilePath := filepath.Join(b.loader.WorkingDir(), filePROFILE)
	if err := os.WriteFile(profilePath, []byte(response), 0644); err != nil {
		return fmt.Errorf("write PROFILE.md: %w", err)
	}
	log.Info("PROFILE.md written", zap.String("path", profilePath))

	if err := b.loader.DeleteBootstrap(); err != nil {
		return err
	}
	return nil
}

func (b *BootstrapRunner) readBootstrap() (string, error) {
	path := filepath.Join(b.loader.WorkingDir(), fileBOOTSTRAP)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
