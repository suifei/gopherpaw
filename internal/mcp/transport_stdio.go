package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

// StdioTransport implements Transport using subprocess stdin/stdout.
type StdioTransport struct {
	cmd    string
	args   []string
	env    map[string]string
	cwd    string

	process *exec.Cmd
	stdin   *bufio.Writer
	stdout  *bufio.Scanner
	mu      sync.Mutex
	running bool
}

// NewStdioTransport creates a new stdio transport from config.
func NewStdioTransport(cfg config.MCPServerConfig) *StdioTransport {
	return &StdioTransport{
		cmd:  cfg.Command,
		args: cfg.Args,
		env:  cfg.Env,
		cwd:  cfg.Cwd,
	}
}

// Start launches the subprocess and establishes stdin/stdout pipes.
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return nil
	}

	t.process = exec.CommandContext(ctx, t.cmd, t.args...)
	t.process.Env = envToSlice(t.env)
	if t.cwd != "" {
		t.process.Dir = t.cwd
	}
	t.process.Stderr = nil

	stdin, err := t.process.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := t.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := t.process.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	t.stdin = bufio.NewWriter(stdin)
	t.stdout = bufio.NewScanner(stdout)
	t.running = true
	return nil
}

// Stop terminates the subprocess.
func (t *StdioTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return nil
	}
	t.running = false
	if t.process != nil && t.process.Process != nil {
		_ = t.process.Process.Kill()
	}
	t.process = nil
	return nil
}

// Call sends a JSON-RPC request and waits for the response.
func (t *StdioTransport) Call(ctx context.Context, req jsonRPCRequest, result interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return fmt.Errorf("transport not running")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := t.stdin.Write(data); err != nil {
		return err
	}
	if err := t.stdin.WriteByte('\n'); err != nil {
		return err
	}
	if err := t.stdin.Flush(); err != nil {
		return err
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	done := make(chan struct{})
	var scanErr error
	go func() {
		if t.stdout.Scan() {
			scanErr = json.Unmarshal(t.stdout.Bytes(), result)
		} else {
			scanErr = t.stdout.Err()
			if scanErr == nil {
				scanErr = fmt.Errorf("unexpected EOF")
			}
		}
		close(done)
	}()

	select {
	case <-done:
		return scanErr
	case <-time.After(time.Until(deadline)):
		return fmt.Errorf("timeout waiting for MCP response")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WriteNotification sends a notification (no response expected).
func (t *StdioTransport) WriteNotification(msg map[string]any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	t.stdin.Write(data)
	t.stdin.WriteByte('\n')
	return t.stdin.Flush()
}

// IsRunning returns true if the transport is active.
func (t *StdioTransport) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

func envToSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

var _ Transport = (*StdioTransport)(nil)
