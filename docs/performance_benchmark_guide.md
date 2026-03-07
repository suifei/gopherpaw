# GopherPaw 性能基准测试指南

> 本文档提供 GopherPaw 性能基准测试的详细说明和最佳实践

## 概述

性能基准测试是确保 GopherPaw 在各种负载下保持良好性能的关键。本文档涵盖：

- 基准测试运行方法
- 性能指标分析
- 优化建议
- 持续性能监控

## 快速开始

### 运行完整基准测试

```bash
# 赋予执行权限
chmod +x scripts/benchmark.sh

# 运行所有基准测试
./scripts/benchmark.sh all

# 查看报告
cat benchmark_reports/benchmark_*.txt
```

### 快速测试

```bash
# 只测试主要包（更快）
./scripts/benchmark.sh quick
```

## 基准测试类型

### 1. Agent 包基准测试

测试 Agent 核心性能：

```bash
go test -bench=. -benchmem ./internal/agent/
```

**关键指标**：
- ReAct 循环执行时间
- 消息处理吞吐量
- 工具调用延迟
- 会话管理开销

### 2. Channels 包基准测试

测试各渠道性能：

```bash
go test -bench=. -benchmem ./internal/channels/
```

**关键指标**：
- 消息发送延迟
- 消息队列吞吐量
- 并发处理能力
- 内存使用

### 3. LLM 包基准测试

测试 LLM 接口性能：

```bash
go test -bench=. -benchmem ./internal/llm/
```

**关键指标**：
- 请求序列化/反序列化
- 消息格式化
- 流式响应处理
- 并发请求

### 4. Memory 包基准测试

测试记忆系统性能：

```bash
go test -bench=. -benchmem ./internal/memory/
```

**关键指标**：
- 记忆存储延迟
- 检索速度（BM25/向量）
- 压缩效率
- Embedding 缓存命中率

### 5. Tools 包基准测试

测试工具执行性能：

```bash
go test -bench=. -benchmem ./internal/tools/
```

**关键指标**：
- 文件 I/O 速度
- 搜索性能
- Web 请求延迟
- 浏览器自动化性能

### 6. MCP 包基准测试

测试 MCP 传输性能：

```bash
go test -bench=. -benchmem ./internal/mcp/
```

**关键指标**：
- JSON-RPC 处理
- 传输延迟（Stdio/HTTP/SSE）
- 并发连接
- 熔断器响应

## 性能分析

### 内存分析

```bash
# 运行内存分析
./scripts/benchmark.sh memory

# 或手动分析特定包
go test -bench=. -benchmem -memprofile=mem.prof ./internal/agent/
go tool pprof -http=:8080 mem.prof
```

**关注点**：
- 内存分配次数
- 内存使用峰值
- GC 压力
- 内存泄漏

### CPU 分析

```bash
# 运行 CPU 分析
./scripts/benchmark.sh cpu

# 或手动分析特定包
go test -bench=. -cpuprofile=cpu.prof ./internal/agent/
go tool pprof -http=:8080 cpu.prof
```

**关注点**：
- 热点函数
- CPU 时间分布
- 优化机会

### 竞态检测

```bash
# 运行竞态检测
./scripts/benchmark.sh race

# 或手动检测
go test -race ./...
```

**关注点**：
- 数据竞争
- 锁争用
- 死锁风险

## 性能基准

### Agent 性能

| 指标 | 目标值 | 说明 |
|------|--------|------|
| ReAct 循环 | < 100ms | 单次推理循环（不含 LLM 调用） |
| 工具调用 | < 10ms | 工具调度和参数解析 |
| 会话创建 | < 5ms | 新会话初始化 |
| 消息处理 | > 1000 msg/s | 消息处理吞吐量 |

### Channels 性能

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 消息发送 | < 50ms | 单条消息发送延迟 |
| 队列吞吐 | > 5000 msg/s | 消息队列处理能力 |
| 并发连接 | > 1000 | 同时支持的连接数 |
| 内存使用 | < 100MB | 单渠道内存占用 |

### LLM 性能

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 请求序列化 | < 1ms | 请求 JSON 序列化 |
| 响应解析 | < 5ms | 响应 JSON 解析 |
| 流式处理 | 实时 | 流式响应无明显延迟 |
| 并发请求 | > 100 | 同时处理的请求数 |

### Memory 性能

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 存储延迟 | < 10ms | 单条记忆存储 |
| 检索速度 | < 50ms | BM25 检索（1000 条记忆） |
| 向量检索 | < 100ms | 向量相似度搜索 |
| 压缩速度 | > 100 KB/s | 记忆压缩吞吐量 |

### Tools 性能

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 文件读取 | > 100 MB/s | 文件读取速度 |
| 文件写入 | > 50 MB/s | 文件写入速度 |
| 搜索速度 | > 1000 files/s | 文件搜索速度 |
| Web 请求 | < 5s | 单次 HTTP 请求（含网络） |

### MCP 性能

| 指标 | 目标值 | 说明 |
|------|--------|------|
| JSON-RPC | < 1ms | 单次 JSON-RPC 调用 |
| Stdio 传输 | < 5ms | Stdio 往返延迟 |
| HTTP 传输 | < 50ms | HTTP 往返延迟 |
| SSE 传输 | < 100ms | SSE 事件延迟 |

## 优化建议

### 1. 减少内存分配

```go
// 不好的做法
func processMessages(msgs []Message) []Result {
    var results []Result
    for _, msg := range msgs {
        results = append(results, process(msg))
    }
    return results
}

// 好的做法
func processMessages(msgs []Message) []Result {
    results := make([]Result, 0, len(msgs))
    for _, msg := range msgs {
        results = append(results, process(msg))
    }
    return results
}
```

### 2. 使用对象池

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processWithPool() string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // 使用 buf 处理
    return buf.String()
}
```

### 3. 避免不必要的锁

```go
// 使用读写锁
type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func (c *Cache) Get(key string) string {
    c.mu.RLock()  // 读操作用读锁
    defer c.mu.RUnlock()
    return c.data[key]
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()  // 写操作用写锁
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### 4. 批量处理

```go
// 批量处理消息
func batchProcess(messages []Message) error {
    batches := makeBatches(messages, 100)
    
    var wg sync.WaitGroup
    errChan := make(chan error, len(batches))
    
    for _, batch := range batches {
        wg.Add(1)
        go func(b []Message) {
            defer wg.Done()
            if err := processBatch(b); err != nil {
                errChan <- err
            }
        }(batch)
    }
    
    wg.Wait()
    close(errChan)
    
    // 处理错误
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### 5. 使用 Context 控制超时

```go
func callLLMWithTimeout(ctx context.Context, req *Request) (*Response, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    return llmClient.Call(ctx, req)
}
```

## 持续性能监控

### 1. CI/CD 集成

```yaml
# .github/workflows/benchmark.yml
name: Performance Benchmark

on:
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * 0'  # 每周运行一次

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Run benchmarks
        run: |
          chmod +x scripts/benchmark.sh
          ./scripts/benchmark.sh quick
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: benchmark-results
          path: benchmark_reports/
```

### 2. 性能回归检测

```bash
# 保存基准结果
go test -bench=. ./... > baseline.txt

# 比较新结果
go test -bench=. ./... > current.txt
benchstat baseline.txt current.txt
```

### 3. 性能监控仪表板

使用 Grafana + Prometheus 监控生产环境性能：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'gopherpaw'
    static_configs:
      - targets: ['localhost:8081']
```

## 性能测试最佳实践

1. **隔离测试环境** - 在专用机器上运行基准测试
2. **多次运行** - 至少运行 3 次取平均值
3. **预热** - 运行前预热 JIT 编译器
4. **关闭节能** - 禁用 CPU 频率调节
5. **固定 CPU** - 使用 `taskset` 固定 CPU 核心
6. **监控资源** - 监控 CPU、内存、I/O 使用
7. **记录环境** - 记录测试环境的硬件和软件配置
8. **版本对比** - 定期对比不同版本的性能

## 故障排查

### 性能下降

1. 检查最近的代码变更
2. 运行 CPU 和内存分析
3. 检查是否有内存泄漏
4. 验证外部依赖性能

### 内存泄漏

1. 使用 pprof 分析内存
2. 检查 goroutine 泄漏
3. 验证资源释放
4. 检查全局变量

### CPU 过高

1. 运行 CPU 分析
2. 检查热点函数
3. 优化算法复杂度
4. 减少不必要的计算

## 相关文档

- [渠道测试指南](./channel_testing_guide.md)
- [MCP 传输测试指南](./mcp_transport_testing_guide.md)
- [架构规范](./architecture_spec.md)
- [API 规范](./api_spec.md)
