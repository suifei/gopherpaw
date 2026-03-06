package scheduler

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

// mockHeartbeatDispatcher 实现 HeartbeatDispatcher 接口供测试使用
type mockHeartbeatDispatcher struct {
	lastCh     string
	lastUserID string
	lastSID    string
	sendFunc   func(ctx context.Context, channel, to, text string) error
	lastFunc   func() (string, string, string)
	sendCalls  int
	mu         sync.Mutex
}

func (m *mockHeartbeatDispatcher) Send(ctx context.Context, channel, to, text string) error {
	m.mu.Lock()
	m.sendCalls++
	m.mu.Unlock()
	if m.sendFunc != nil {
		return m.sendFunc(ctx, channel, to, text)
	}
	return nil
}

func (m *mockHeartbeatDispatcher) LastDispatch() (channel, userID, sessionID string) {
	if m.lastFunc != nil {
		return m.lastFunc()
	}
	return m.lastCh, m.lastUserID, m.lastSID
}

// TestHeartbeatRunner_NewHeartbeatRunner 测试 NewHeartbeatRunner
func TestHeartbeatRunner_NewHeartbeatRunner(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "last",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	runner := NewHeartbeatRunner(ag, nil, cfg, dispatcher)

	if runner == nil {
		t.Errorf("NewHeartbeatRunner returned nil")
	}
	if runner.agent != ag {
		t.Errorf("agent not set correctly")
	}
	if runner.config != cfg {
		t.Errorf("config not set correctly")
	}
}

// TestHeartbeatRunner_Start_EmptyEvery 测试空 Every 的 Start
func TestHeartbeatRunner_Start_EmptyEvery(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	runner := NewHeartbeatRunner(ag, nil, cfg, dispatcher)
	ctx := context.Background()

	err := runner.Start(ctx)
	if err != nil {
		t.Errorf("Start with empty every returned error: %v", err)
	}
}

// TestHeartbeatRunner_Start_InvalidEvery 测试无效 Every 的 Start
func TestHeartbeatRunner_Start_InvalidEvery(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "invalid",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	// "invalid" 会被视为 0 duration，然后默认为 30m，所以不会返回错误
	err := runner.Start(ctx)
	if err != nil {
		t.Errorf("Start with invalid every should not return error (defaults to 30m): %v", err)
	}
	runner.Stop()
}

// TestHeartbeatRunner_Start_AlreadyRunning 测试已运行状态下重复 Start
func TestHeartbeatRunner_Start_AlreadyRunning(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	err := runner.Start(ctx)
	if err != nil {
		t.Fatalf("First Start returned error: %v", err)
	}
	defer runner.Stop()

	// 再次调用 Start，应该返回 nil
	err = runner.Start(ctx)
	if err != nil {
		t.Errorf("Second Start returned error: %v", err)
	}
}

// TestHeartbeatRunner_Stop 测试 Stop
func TestHeartbeatRunner_Stop(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	err := runner.Start(ctx)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	err = runner.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}

	// 再次调用 Stop 应该安全
	err = runner.Stop()
	if err != nil {
		t.Errorf("Second Stop returned error: %v", err)
	}
}

// TestHeartbeatRunner_Stop_WithoutStart 测试未启动状态下的 Stop
func TestHeartbeatRunner_Stop_WithoutStart(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	runner := NewHeartbeatRunner(ag, nil, cfg, dispatcher)

	err := runner.Stop()
	if err != nil {
		t.Errorf("Stop without Start returned error: %v", err)
	}
}

// TestHeartbeatRunner_WithNilLoader 测试 loader 为 nil 的情况
func TestHeartbeatRunner_WithNilLoader(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	runner := NewHeartbeatRunner(ag, nil, cfg, dispatcher)

	// 即使 loader 为 nil，Stop 也应该成功
	err := runner.Stop()
	if err != nil {
		t.Errorf("Stop on runner with nil loader returned error: %v", err)
	}
}

// TestParseTime 测试 parseTime 函数
func TestParseTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:  "valid time",
			input: "10:30",
			want:  630, // 10*60 + 30
		},
		{
			name:  "midnight",
			input: "00:00",
			want:  0,
		},
		{
			name:  "end of day",
			input: "23:59",
			want:  1439, // 23*60 + 59
		},
		{
			name:    "invalid format",
			input:   "10-30",
			wantErr: true,
		},
		{
			name:    "missing colon",
			input:   "1030",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too many parts",
			input:   "10:30:45",
			wantErr: true,
		},
		{
			name:  "single digit hour",
			input: "9:05",
			want:  545, // 9*60 + 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTime error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseTime got = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestParseEveryToDuration 测试 parseEveryToDuration 函数
func TestParseEveryToDuration(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{
			name:  "30 minutes",
			input: "30m",
			want:  30 * time.Minute,
		},
		{
			name:  "1 hour",
			input: "1h",
			want:  1 * time.Hour,
		},
		{
			name:  "2 hours 30 minutes",
			input: "2h30m",
			want:  2*time.Hour + 30*time.Minute,
		},
		{
			name:  "60 seconds",
			input: "60s",
			want:  60 * time.Second,
		},
		{
			name:  "combined",
			input: "1h30m45s",
			want:  1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:  "empty string defaults to 30m",
			input: "",
			want:  30 * time.Minute,
		},
		{
			name:  "invalid format defaults to 30m",
			input: "invalid",
			want:  30 * time.Minute,
		},
		{
			name:  "zero value",
			input: "0m",
			want:  30 * time.Minute, // zero 时默认为 30m
		},
		{
			name:  "only hours",
			input: "3h",
			want:  3 * time.Hour,
		},
		{
			name:  "only seconds",
			input: "120s",
			want:  120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEveryToDuration(tt.input)
			if got != tt.want {
				t.Errorf("parseEveryToDuration got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseEveryToCron 测试 parseEveryToCron 函数
func TestParseEveryToCron(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "30 minutes",
			input: "30m",
			want:  "*/30 * * * *",
		},
		{
			name:  "1 hour",
			input: "1h",
			want:  "0 */1 * * *",
		},
		{
			name:  "2 hours",
			input: "2h",
			want:  "0 */2 * * *",
		},
		{
			name:  "1 hour 30 minutes",
			input: "1h30m",
			want:  "30 */1 * * *",
		},
		{
			name:  "45 minutes",
			input: "45m",
			want:  "*/45 * * * *",
		},
		{
			name:  "3 hours",
			input: "3h",
			want:  "0 */3 * * *",
		},
		{
			name:  "90 minutes (1.5 hours)",
			input: "90m",
			want:  "30 */1 * * *",
		},
		{
			name:    "invalid defaults to 30m",
			input:   "invalid",
			want:    "*/30 * * * *",
			wantErr: false,
		},
		{
			name:    "empty defaults to 30m",
			input:   "",
			want:    "*/30 * * * *",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEveryToCron(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEveryToCron error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseEveryToCron got = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestHeartbeatRunner_InActiveHours 测试 inActiveHours 方法
func TestHeartbeatRunner_InActiveHours(t *testing.T) {
	tests := []struct {
		name        string
		activeHours *config.ActiveHours
		want        bool
	}{
		{
			name:        "no active hours config",
			activeHours: nil,
			want:        true,
		},
		{
			name:        "empty active hours",
			activeHours: &config.ActiveHours{Start: "", End: ""},
			want:        true,
		},
		{
			name:        "only start time",
			activeHours: &config.ActiveHours{Start: "09:00", End: ""},
			want:        true, // Invalid, defaults to true
		},
		{
			name:        "invalid time format",
			activeHours: &config.ActiveHours{Start: "invalid", End: "18:00"},
			want:        true, // Invalid, defaults to true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag := &mockAgent{}
			cfg := config.HeartbeatConfig{
				Every:       "30m",
				Target:      "",
				ActiveHours: tt.activeHours,
			}
			runner := NewHeartbeatRunner(ag, nil, cfg, nil)

			got := runner.inActiveHours()
			if got != tt.want {
				t.Errorf("inActiveHours got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHeartbeatRunner_ConcurrentStartStop 测试并发的 Start 和 Stop
func TestHeartbeatRunner_ConcurrentStartStop(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make([]error, 0)
	var mu sync.Mutex

	// 并发调用 Start
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runner.Start(ctx); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		t.Errorf("Concurrent Start returned errors: %v", errors)
	}

	// 并发调用 Stop
	errors = errors[:0]
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runner.Stop(); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		t.Errorf("Concurrent Stop returned errors: %v", errors)
	}
}

// TestHeartbeatRunner_ContextCancellation 测试 context 取消
func TestHeartbeatRunner_ContextCancellation(t *testing.T) {
	ag := &mockAgent{}
	cfg := config.HeartbeatConfig{
		Every:  "30m",
		Target: "",
	}
	dispatcher := &mockHeartbeatDispatcher{}

	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx, cancel := context.WithCancel(context.Background())

	err := runner.Start(ctx)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	// 立即取消 context
	cancel()

	// 等待一下确保 context 被处理
	time.Sleep(100 * time.Millisecond)

	// Stop 应该正常工作
	err = runner.Stop()
	if err != nil {
		t.Errorf("Stop after context cancellation returned error: %v", err)
	}
}

// TestHeartbeatRunner_EdgeCases 测试边界情况
func TestHeartbeatRunner_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		every   string
		wantErr bool
	}{
		{
			name:    "zero duration defaults to 30m",
			every:   "0h",
			wantErr: false,
		},
		{
			name:    "valid 5 minutes",
			every:   "5m",
			wantErr: false,
		},
		{
			name:    "large duration",
			every:   "24h",
			wantErr: false,
		},
		{
			name:    "mixed large values",
			every:   "5h30m45s",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag := &mockAgent{}
			cfg := config.HeartbeatConfig{
				Every:  tt.every,
				Target: "",
			}
			tmpDir := t.TempDir()
			loader := agent.NewPromptLoader(tmpDir, "")
			runner := NewHeartbeatRunner(ag, loader, cfg, nil)
			ctx := context.Background()

			err := runner.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				runner.Stop()
			}
		})
	}
}

// TestCronScheduler_AddJobDuplicate 测试重复添加相同名称的 job
func TestCronScheduler_AddJobDuplicate(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sched.Stop(ctx)

	job1 := &mockJob{name: "duplicate-job"}
	if err := sched.AddJob("* * * * *", job1); err != nil {
		t.Fatalf("First AddJob: %v", err)
	}

	// 添加相同名称的 job 应该返回错误
	job2 := &mockJob{name: "duplicate-job"}
	err := sched.AddJob("* * * * *", job2)
	if err == nil {
		t.Errorf("AddJob with duplicate name should return error")
	}
}

// TestCronScheduler_InvalidSpec 测试无效的 cron 规范
func TestCronScheduler_InvalidSpec(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sched.Stop(ctx)

	job := &mockJob{name: "invalid-spec-job"}
	err := sched.AddJob("invalid spec", job)
	if err == nil {
		t.Errorf("AddJob with invalid spec should return error")
	}
}

// TestCronScheduler_Stop_WithContext 测试用 context 的 Stop
func TestCronScheduler_Stop_WithContext(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	err := sched.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

// TestCronScheduler_Stop_WithCancelledContext 测试用已取消 context 的 Stop
func TestCronScheduler_Stop_WithCancelledContext(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sched.Stop(cancelledCtx)
	if err == nil {
		t.Errorf("Stop with cancelled context should return context error")
	}
}

// TestCronScheduler_AddJob_Disabled 测试禁用时添加 job
func TestCronScheduler_AddJob_Disabled(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: false})
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sched.Stop(ctx)

	// 禁用时 AddJob 应该返回 nil（no-op）
	job := &mockJob{name: "disabled-job"}
	err := sched.AddJob("* * * * *", job)
	if err != nil {
		t.Errorf("AddJob on disabled scheduler returned error: %v", err)
	}
}

// TestCronScheduler_Start_WithCancelledContext 测试用已取消 context 的 Start
func TestCronScheduler_Start_WithCancelledContext(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// 即使 context 已取消，Start 也应该成功（goroutine 会立即退出）
	err := sched.Start(cancelledCtx)
	if err != nil {
		t.Errorf("Start with cancelled context returned error: %v", err)
	}

	// 确保能 Stop
	sched.Stop(context.Background())
}

// TestCronScheduler_ConcurrentAddJob 测试并发添加 job
func TestCronScheduler_ConcurrentAddJob(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})
	ctx := context.Background()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sched.Stop(ctx)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// 并发添加多个 job
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			job := &mockJob{name: fmt.Sprintf("job-%d", idx)}
			if err := sched.AddJob("* * * * *", job); err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 10 {
		t.Errorf("Concurrent AddJob: got %d successful adds, want 10", successCount)
	}
}

type mockAgentFail struct {
	failRun bool
}

func (m *mockAgentFail) Run(ctx context.Context, chatID string, message string) (string, error) {
	if m.failRun {
		return "", fmt.Errorf("mock agent run failed")
	}
	return "ok", nil
}

func (m *mockAgentFail) RunStream(ctx context.Context, chatID string, message string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "ok"
	close(ch)
	return ch, nil
}

type mockDispatcherFail struct {
	failSend bool
	sendErr  error
	lastCh   string
	lastUID  string
	lastSID  string
}

func (m *mockDispatcherFail) Send(ctx context.Context, channel, to, text string) error {
	m.lastCh = channel
	m.lastUID = to
	m.lastSID = text
	if m.failSend {
		if m.sendErr != nil {
			return m.sendErr
		}
		return fmt.Errorf("mock dispatcher send failed")
	}
	return nil
}

func (m *mockDispatcherFail) LastDispatch() (channel, userID, sessionID string) {
	return m.lastCh, m.lastUID, m.lastSID
}

func TestHeartbeatRunner_RunOnce_LoaderError(t *testing.T) {
	ag := &mockAgentFail{}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "",
	}
	dispatcher := &mockDispatcherFail{}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}

func TestHeartbeatRunner_RunOnce_EmptyContent(t *testing.T) {
	ag := &mockAgentFail{}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	heartbeatPath := tmpDir + "/HEARTBEAT.md"
	if err := os.WriteFile(heartbeatPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty HEARTBEAT.md: %v", err)
	}
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "",
	}
	dispatcher := &mockDispatcherFail{}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}

func TestHeartbeatRunner_RunOnce_AgentFail(t *testing.T) {
	ag := &mockAgentFail{failRun: true}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	heartbeatPath := tmpDir + "/HEARTBEAT.md"
	if err := os.WriteFile(heartbeatPath, []byte("heartbeat content"), 0644); err != nil {
		t.Fatalf("Failed to write HEARTBEAT.md: %v", err)
	}
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "",
	}
	dispatcher := &mockDispatcherFail{}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}

func TestHeartbeatRunner_RunOnce_TargetLast(t *testing.T) {
	ag := &mockAgentFail{}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	heartbeatPath := tmpDir + "/HEARTBEAT.md"
	if err := os.WriteFile(heartbeatPath, []byte("heartbeat content"), 0644); err != nil {
		t.Fatalf("Failed to write HEARTBEAT.md: %v", err)
	}
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "last",
	}
	dispatcher := &mockDispatcherFail{
		lastCh:  "test-channel",
		lastUID: "user123",
		lastSID: "test-channel:session456",
	}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}

func TestHeartbeatRunner_RunOnce_SendFail(t *testing.T) {
	ag := &mockAgentFail{}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	heartbeatPath := tmpDir + "/HEARTBEAT.md"
	if err := os.WriteFile(heartbeatPath, []byte("heartbeat content"), 0644); err != nil {
		t.Fatalf("Failed to write HEARTBEAT.md: %v", err)
	}
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "last",
	}
	dispatcher := &mockDispatcherFail{
		failSend: true,
		lastCh:   "test-channel",
		lastUID:  "user123",
		lastSID:  "test-channel:session456",
	}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}

func TestHeartbeatRunner_RunOnce_OutsideActiveHours(t *testing.T) {
	ag := &mockAgentFail{}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	heartbeatPath := tmpDir + "/HEARTBEAT.md"
	if err := os.WriteFile(heartbeatPath, []byte("heartbeat content"), 0644); err != nil {
		t.Fatalf("Failed to write HEARTBEAT.md: %v", err)
	}
	now := time.Now()
	hour := (now.Hour() + 12) % 24
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "",
		ActiveHours: &config.ActiveHours{
			Start: fmt.Sprintf("%02d:00", hour),
			End:   fmt.Sprintf("%02d:00", hour),
		},
	}
	dispatcher := &mockDispatcherFail{}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}

func TestHeartbeatRunner_RunOnce_Success(t *testing.T) {
	ag := &mockAgent{}
	tmpDir := t.TempDir()
	loader := agent.NewPromptLoader(tmpDir, "")
	heartbeatPath := tmpDir + "/HEARTBEAT.md"
	if err := os.WriteFile(heartbeatPath, []byte("heartbeat content"), 0644); err != nil {
		t.Fatalf("Failed to write HEARTBEAT.md: %v", err)
	}
	cfg := config.HeartbeatConfig{
		Every:  "1s",
		Target: "",
	}
	dispatcher := &mockDispatcherFail{}

	runner := NewHeartbeatRunner(ag, loader, cfg, dispatcher)
	ctx := context.Background()

	if err := runner.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	defer runner.Stop()

	time.Sleep(2 * time.Second)
}
