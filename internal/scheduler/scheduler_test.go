package scheduler

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

type mockAgent struct{}

func (m *mockAgent) Run(ctx context.Context, chatID string, message string) (string, error) {
	return "ok", nil
}

func (m *mockAgent) RunStream(ctx context.Context, chatID string, message string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "ok"
	close(ch)
	return ch, nil
}

type mockJob struct {
	name string
	run  func(ctx context.Context) error
}

func (j *mockJob) Run(ctx context.Context) error {
	if j.run != nil {
		return j.run(ctx)
	}
	return nil
}

func (j *mockJob) Name() string { return j.name }

func TestCronScheduler_AddJob(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: true})
	ctx := context.Background()
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer sched.Stop(ctx)
	if err := sched.AddJob("* * * * *", &mockJob{name: "test-job"}); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
}

func TestCronScheduler_Disabled(t *testing.T) {
	sched := New(&mockAgent{}, config.SchedulerConfig{Enabled: false})
	ctx := context.Background()
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := sched.AddJob("* * * * *", &mockJob{name: "x"}); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	sched.Stop(ctx)
}
