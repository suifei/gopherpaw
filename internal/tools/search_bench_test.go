package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkGrepSearch_Simple(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(grepArgs{Pattern: "test", Path: tmp, IsRegex: false, CaseSensitive: false})
	tool := &GrepSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGrepSearch_Regex(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(grepArgs{Pattern: "t.st", Path: tmp, IsRegex: true, CaseSensitive: false})
	tool := &GrepSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGrepSearch_CaseSensitive(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("Test Line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(grepArgs{Pattern: "Test", Path: tmp, IsRegex: false, CaseSensitive: true})
	tool := &GrepSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGrepSearch_WithContext(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(grepArgs{Pattern: "test", Path: tmp, IsRegex: false, CaseSensitive: false, ContextLines: 3})
	tool := &GrepSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGrepSearch_MultipleFiles(b *testing.B) {
	tmp := b.TempDir()
	for i := 0; i < 10; i++ {
		file := tmp + "/test.txt"
		content := strings.Repeat("test line\n", 10000)
		os.WriteFile(file, []byte(content), 0644)
	}

	args, _ := json.Marshal(grepArgs{Pattern: "test", Path: tmp, IsRegex: false, CaseSensitive: false})
	tool := &GrepSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGlobSearch_Simple(b *testing.B) {
	tmp := b.TempDir()
	for i := 0; i < 100; i++ {
		file := tmp + "/test.go"
		content := strings.Repeat("test line\n", 100)
		os.WriteFile(file, []byte(content), 0644)
	}

	args, _ := json.Marshal(globArgs{Pattern: "*.go", Path: tmp})
	tool := &GlobSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGlobSearch_Recursive(b *testing.B) {
	tmp := b.TempDir()
	subdir := tmp + "/sub"
	os.Mkdir(subdir, 0755)
	for i := 0; i < 50; i++ {
		file := tmp + "/test.go"
		content := strings.Repeat("test line\n", 100)
		os.WriteFile(file, []byte(content), 0644)
	}
	for i := 0; i < 50; i++ {
		file := subdir + "/test.go"
		content := strings.Repeat("test line\n", 100)
		os.WriteFile(file, []byte(content), 0644)
	}

	args, _ := json.Marshal(globArgs{Pattern: "**/*.go", Path: tmp})
	tool := &GlobSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkGlobSearch_MultipleExtensions(b *testing.B) {
	tmp := b.TempDir()
	for i := 0; i < 100; i++ {
		file := tmp + "/test.go"
		content := strings.Repeat("test line\n", 100)
		os.WriteFile(file, []byte(content), 0644)
	}
	for i := 0; i < 100; i++ {
		file := tmp + "/test.txt"
		content := strings.Repeat("test line\n", 100)
		os.WriteFile(file, []byte(content), 0644)
	}
	for i := 0; i < 100; i++ {
		file := tmp + "/test.json"
		content := strings.Repeat("test line\n", 100)
		os.WriteFile(file, []byte(content), 0644)
	}

	args, _ := json.Marshal(globArgs{Pattern: "*.go", Path: tmp})
	tool := &GlobSearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkMemorySearch_Small(b *testing.B) {
	tmp := b.TempDir()
	workingDir = tmp

	tool := &MemorySearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args, _ := json.Marshal(memorySearchArgs{Query: "test query"})
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkMemorySearch_Medium(b *testing.B) {
	tmp := b.TempDir()
	workingDir = tmp

	vectorsDir := filepath.Join(tmp, "data", "vectors")
	os.MkdirAll(vectorsDir, 0755)

	for i := 0; i < 100; i++ {
		args, _ := json.Marshal(memorySearchArgs{Query: "test query", TopK: 5})
		tool := &MemorySearchTool{}
		tool.Execute(context.Background(), string(args))
	}

	tool := &MemorySearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args, _ := json.Marshal(memorySearchArgs{Query: "test query"})
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkMemorySearch_Large(b *testing.B) {
	tmp := b.TempDir()
	workingDir = tmp

	vectorsDir := filepath.Join(tmp, "data", "vectors")
	os.MkdirAll(vectorsDir, 0755)

	for i := 0; i < 1000; i++ {
		args, _ := json.Marshal(memorySearchArgs{Query: "test query", TopK: 5})
		tool := &MemorySearchTool{}
		tool.Execute(context.Background(), string(args))
	}

	tool := &MemorySearchTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args, _ := json.Marshal(memorySearchArgs{Query: "test query"})
		tool.Execute(context.Background(), string(args))
	}
}
