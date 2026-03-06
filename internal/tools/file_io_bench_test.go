package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func BenchmarkReadFile_Small(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 100)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(readFileArgs{FilePath: file})
	tool := &ReadFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkReadFile_Medium(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(readFileArgs{FilePath: file})
	tool := &ReadFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkReadFile_Large(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 100000)
	os.WriteFile(file, []byte(content), 0644)

	args, _ := json.Marshal(readFileArgs{FilePath: file})
	tool := &ReadFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkReadFile_Range(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("test line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	startLine := 100
	endLine := 200
	args, _ := json.Marshal(readFileArgs{FilePath: file, StartLine: &startLine, EndLine: &endLine})
	tool := &ReadFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkWriteFile_Small(b *testing.B) {
	tmp := b.TempDir()
	tool := &WriteFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := tmp + "/test.txt"
		content := strings.Repeat("test line\n", 100)
		args, _ := json.Marshal(writeFileArgs{FilePath: file, Content: content})
		tool.Execute(context.Background(), string(args))
		os.Remove(file)
	}
}

func BenchmarkWriteFile_Medium(b *testing.B) {
	tmp := b.TempDir()
	tool := &WriteFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := tmp + "/test.txt"
		content := strings.Repeat("test line\n", 10000)
		args, _ := json.Marshal(writeFileArgs{FilePath: file, Content: content})
		tool.Execute(context.Background(), string(args))
		os.Remove(file)
	}
}

func BenchmarkWriteFile_Large(b *testing.B) {
	tmp := b.TempDir()
	tool := &WriteFileTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := tmp + "/test.txt"
		content := strings.Repeat("test line\n", 100000)
		args, _ := json.Marshal(writeFileArgs{FilePath: file, Content: content})
		tool.Execute(context.Background(), string(args))
		os.Remove(file)
	}
}

func BenchmarkEditFile_Small(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("old line\n", 100)
	os.WriteFile(file, []byte(content), 0644)

	tool := &EditFileTool{}
	args, _ := json.Marshal(editFileArgs{FilePath: file, OldText: "old line", NewText: "new line"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkEditFile_Medium(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("old line\n", 10000)
	os.WriteFile(file, []byte(content), 0644)

	tool := &EditFileTool{}
	args, _ := json.Marshal(editFileArgs{FilePath: file, OldText: "old line", NewText: "new line"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkEditFile_Large(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	content := strings.Repeat("old line\n", 100000)
	os.WriteFile(file, []byte(content), 0644)

	tool := &EditFileTool{}
	args, _ := json.Marshal(editFileArgs{FilePath: file, OldText: "old line", NewText: "new line"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkAppendFile_Small(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	os.WriteFile(file, []byte("initial\n"), 0644)

	tool := &AppendFileTool{}
	content := strings.Repeat("new line\n", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args, _ := json.Marshal(appendFileArgs{FilePath: file, Content: content})
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkAppendFile_Medium(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	os.WriteFile(file, []byte("initial\n"), 0644)

	tool := &AppendFileTool{}
	content := strings.Repeat("new line\n", 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args, _ := json.Marshal(appendFileArgs{FilePath: file, Content: content})
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkAppendFile_Large(b *testing.B) {
	tmp := b.TempDir()
	file := tmp + "/test.txt"
	os.WriteFile(file, []byte("initial\n"), 0644)

	tool := &AppendFileTool{}
	content := strings.Repeat("new line\n", 100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args, _ := json.Marshal(appendFileArgs{FilePath: file, Content: content})
		tool.Execute(context.Background(), string(args))
	}
}
