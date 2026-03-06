package scheduler

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func BenchmarkHeartbeatRunner_RunOnce(b *testing.B) {
	tmp := b.TempDir()
	loader := agent.NewPromptLoader(tmp, "")

	runner := NewHeartbeatRunner(&mockAgent{}, loader, config.HeartbeatConfig{}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.runOnce(context.Background())
	}
}

func BenchmarkHeartbeatRunner_RunOnce_WithLongResponse(b *testing.B) {
	tmp := b.TempDir()
	loader := agent.NewPromptLoader(tmp, "")

	longResponse := string(make([]byte, 10000))
	for i := range longResponse {
		longResponse = longResponse[:i] + "a" + longResponse[i+1:]
	}

	runner := NewHeartbeatRunner(&mockAgent{}, loader, config.HeartbeatConfig{}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.runOnce(context.Background())
	}
}

func BenchmarkParseEveryToDuration_Simple(b *testing.B) {
	every := "30m"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToDuration(every)
	}
}

func BenchmarkParseEveryToDuration_Complex(b *testing.B) {
	every := "2h30m15s"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToDuration(every)
	}
}

func BenchmarkParseEveryToDuration_Hours(b *testing.B) {
	every := "24h"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToDuration(every)
	}
}

func BenchmarkParseEveryToDuration_Minutes(b *testing.B) {
	every := "60m"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToDuration(every)
	}
}

func BenchmarkParseEveryToDuration_Seconds(b *testing.B) {
	every := "3600s"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToDuration(every)
	}
}

func BenchmarkParseEveryToCron_Minutes(b *testing.B) {
	every := "30m"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToCron(every)
	}
}

func BenchmarkParseEveryToCron_Hours(b *testing.B) {
	every := "2h"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToCron(every)
	}
}

func BenchmarkParseEveryToCron_Complex(b *testing.B) {
	every := "2h30m"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseEveryToCron(every)
	}
}

func BenchmarkParseTime(b *testing.B) {
	timeStr := "14:30"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseTime(timeStr)
	}
}

func BenchmarkInActiveHours_WithinRange(b *testing.B) {
	runner := NewHeartbeatRunner(nil, nil, config.HeartbeatConfig{
		ActiveHours: &config.ActiveHours{
			Start: "09:00",
			End:   "18:00",
		},
	}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.inActiveHours()
	}
}

func BenchmarkInActiveHours_OutsideRange(b *testing.B) {
	runner := NewHeartbeatRunner(nil, nil, config.HeartbeatConfig{
		ActiveHours: &config.ActiveHours{
			Start: "09:00",
			End:   "18:00",
		},
	}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.inActiveHours()
	}
}
