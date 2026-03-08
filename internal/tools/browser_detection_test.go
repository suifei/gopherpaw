package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFindChromeExecutable 测试浏览器路径检测功能
func TestFindChromeExecutable(t *testing.T) {
	// 保存原始环境变量
	origEnv := os.Getenv(chromeBinEnv)
	defer func() {
		if origEnv == "" {
			os.Unsetenv(chromeBinEnv)
		} else {
			os.Setenv(chromeBinEnv, origEnv)
		}
	}()

	t.Run("environment variable takes precedence", func(t *testing.T) {
		// 创建临时文件模拟可执行文件
		tmpDir := t.TempDir()
		fakeChrome := filepath.Join(tmpDir, "fake-chrome")
		if err := os.WriteFile(fakeChrome, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}

		os.Setenv(chromeBinEnv, fakeChrome)
		result := findChromeExecutable()
		if result != fakeChrome {
			t.Errorf("expected %s, got %s", fakeChrome, result)
		}
	})

	t.Run("invalid environment variable is ignored", func(t *testing.T) {
		os.Setenv(chromeBinEnv, "/nonexistent/chrome")
		result := findChromeExecutable()
		// 应该回退到系统扫描或返回空字符串
		if result == "/nonexistent/chrome" {
			t.Error("should not return non-existent path from env")
		}
	})

	t.Run("empty environment variable falls back to system scan", func(t *testing.T) {
		os.Unsetenv(chromeBinEnv)
		result := findChromeExecutable()
		// 结果取决于系统是否有浏览器
		// 只验证不崩溃即可
		_ = result
	})
}

// TestChromePaths 测试各平台的路径定义是否完整
func TestChromePaths(t *testing.T) {
	t.Run("linux paths defined", func(t *testing.T) {
		paths, ok := chromePaths["linux"]
		if !ok {
			t.Fatal("linux paths not defined")
		}
		if len(paths) == 0 {
			t.Error("no linux paths defined")
		}
		// 验证一些关键路径
		expectedPaths := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
		for _, expected := range expectedPaths {
			found := false
			for _, p := range paths {
				if p == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected path %s not found in chromePaths", expected)
			}
		}
	})

	t.Run("darwin paths defined", func(t *testing.T) {
		paths, ok := chromePaths["darwin"]
		if !ok {
			t.Fatal("darwin paths not defined")
		}
		if len(paths) == 0 {
			t.Error("no darwin paths defined")
		}
	})

	t.Run("windows paths defined", func(t *testing.T) {
		paths, ok := chromePaths["windows"]
		if !ok {
			t.Fatal("windows paths not defined")
		}
		if len(paths) == 0 {
			t.Error("no windows paths defined")
		}
	})

	t.Run("current platform has paths", func(t *testing.T) {
		paths, ok := chromePaths[runtime.GOOS]
		if !ok {
			t.Skipf("platform %s not tested", runtime.GOOS)
		}
		if len(paths) == 0 {
			t.Errorf("no paths defined for current platform %s", runtime.GOOS)
		}
	})
}

// TestIsRunningInContainer 测试容器环境检测
func TestIsRunningInContainer(t *testing.T) {
	// 保存原始环境变量
	origEnv := os.Getenv(runningInContainerEnv)
	defer func() {
		if origEnv == "" {
			os.Unsetenv(runningInContainerEnv)
		} else {
			os.Setenv(runningInContainerEnv, origEnv)
		}
	}()

	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"env variable true", "true", true},
		{"env variable True", "True", true},
		{"env variable TRUE", "TRUE", true},
		{"env variable 1", "1", true},
		{"env variable yes", "yes", true},
		{"env variable Yes", "Yes", true},
		{"env variable 0", "0", false},
		{"env variable false", "false", false},
		{"env variable empty", "", false},
		{"env variable random", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv(runningInContainerEnv)
			} else {
				os.Setenv(runningInContainerEnv, tt.envValue)
			}
			got := isRunningInContainer()
			if got != tt.want {
				t.Errorf("isRunningInContainer() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBrowserLaunchOptions 测试浏览器启动选项构建
func TestBrowserLaunchOptions(t *testing.T) {
	t.Run("findChromeExecutable does not crash", func(t *testing.T) {
		path := findChromeExecutable()
		// 结果取决于系统配置，只验证不崩溃
		t.Logf("detected chrome path: %q", path)
	})

	t.Run("isRunningInContainer does not crash", func(t *testing.T) {
		inContainer := isRunningInContainer()
		t.Logf("running in container: %v", inContainer)
	})
}

// BenchmarkFindChromeExecutable 性能测试
func BenchmarkFindChromeExecutable(b *testing.B) {
	// 确保环境变量未设置，以测试系统扫描性能
	os.Unsetenv(chromeBinEnv)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = findChromeExecutable()
	}
}

// BenchmarkIsRunningInContainer 性能测试
func BenchmarkIsRunningInContainer(b *testing.B) {
	os.Unsetenv(runningInContainerEnv)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = isRunningInContainer()
	}
}
