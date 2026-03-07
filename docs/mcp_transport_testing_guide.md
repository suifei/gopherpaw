# GopherPaw MCP 传输测试指南

> 本文档提供 GopherPaw MCP (Model Context Protocol) 传输层的测试方法和最佳实践

## 概述

GopherPaw 支持三种 MCP 传输方式：

- **Stdio Transport** - 标准输入/输出传输（最常用）
- **HTTP Transport** - HTTP 传输
- **SSE Transport** - Server-Sent Events 传输

## MCP 协议基础

MCP 是一个用于 AI 模型上下文管理的协议，支持：

- **工具调用** - Agent 可以调用外部工具
- **资源访问** - 访问文件、数据库等资源
- **提示词管理** - 动态加载提示词模板
- **会话管理** - 管理对话上下文

## 传输类型详解

### 1. Stdio Transport

**适用场景**：
- 本地 MCP 服务器（Python/Node.js 脚本）
- 快速原型开发
- 简单的命令行工具

**配置示例**：

```yaml
mcp:
  servers:
    my-mcp-server:
      name: "My MCP Server"
      description: "A simple MCP server"
      transport: "stdio"
      command: "python"
      args: ["mcp_server.py"]
      cwd: "/path/to/server"
      env:
        API_KEY: "your-api-key"
      enabled: true
```

**测试方法**：

```bash
# 1. 创建简单的 MCP 服务器脚本
cat > test_mcp_server.py << 'EOF'
import json
import sys

def handle_request(request):
    method = request.get("method")
    
    if method == "initialize":
        return {
            "jsonrpc": "2.0",
            "id": request["id"],
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {}
            }
        }
    
    return {
        "jsonrpc": "2.0",
        "id": request["id"],
        "error": {"code": -32601, "message": "Method not found"}
    }

for line in sys.stdin:
    try:
        request = json.loads(line)
        response = handle_request(request)
        print(json.dumps(response), flush=True)
    except Exception as e:
        error_response = {
            "jsonrpc": "2.0",
            "id": None,
            "error": {"code": -32700, "message": str(e)}
        }
        print(json.dumps(error_response), flush=True)
EOF

# 2. 运行测试
go test ./internal/mcp/ -run TestStdioTransport -v

# 3. 集成测试
./gopherpaw app
```

### 2. HTTP Transport

**适用场景**：
- 远程 MCP 服务器
- 需要 HTTP API 的场景
- 复杂的 MCP 服务

**配置示例**：

```yaml
mcp:
  servers:
    remote-mcp:
      name: "Remote MCP Server"
      description: "HTTP-based MCP server"
      transport: "http"
      url: "http://localhost:8080/mcp"
      headers:
        Authorization: "Bearer your-token"
        X-API-Key: "your-api-key"
      enabled: true
```

**测试方法**：

```bash
# 1. 启动测试 HTTP 服务器（可以使用 Node.js 快速搭建）
cat > test_http_server.js << 'EOF'
const http = require('http');

const server = http.createServer((req, res) => {
  if (req.method === 'POST' && req.url === '/mcp') {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', () => {
      const request = JSON.parse(body);
      
      if (request.method === 'initialize') {
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({
          jsonrpc: '2.0',
          id: request.id,
          result: {
            protocolVersion: '2024-11-05',
            capabilities: {}
          }
        }));
      } else {
        res.writeHead(200, {'Content-Type': 'application/json'});
        res.end(JSON.stringify({
          jsonrpc: '2.0',
          id: request.id,
          error: {code: -32601, message: 'Method not found'}
        }));
      }
    });
  }
});

server.listen(8080, () => console.log('Test MCP server running on port 8080'));
EOF

node test_http_server.js &

# 2. 运行测试
go test ./internal/mcp/ -run TestHTTPTransport -v

# 3. 清理
pkill -f test_http_server.js
```

### 3. SSE Transport

**适用场景**：
- 需要服务器推送的场景
- 实时更新
- 长连接场景

**配置示例**：

```yaml
mcp:
  servers:
    sse-mcp:
      name: "SSE MCP Server"
      description: "SSE-based MCP server"
      transport: "sse"
      url: "http://localhost:8081/mcp"
      headers:
        Authorization: "Bearer your-token"
      enabled: true
```

**测试方法**：

```bash
# 1. 创建 SSE 测试服务器
cat > test_sse_server.js << 'EOF'
const http = require('http');

const server = http.createServer((req, res) => {
  if (req.method === 'POST' && req.url === '/mcp') {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', () => {
      const request = JSON.parse(body);
      
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive'
      });
      
      const response = {
        jsonrpc: '2.0',
        id: request.id,
        result: {
          protocolVersion: '2024-11-05',
          capabilities: {}
        }
      };
      
      res.write(`data: ${JSON.stringify(response)}\n\n`);
      res.end();
    });
  }
});

server.listen(8081, () => console.log('SSE test server running on port 8081'));
EOF

node test_sse_server.js &

# 2. 运行测试
go test ./internal/mcp/ -run TestSSETransport -v

# 3. 清理
pkill -f test_sse_server.js
```

## 测试类型

### 1. 单元测试

测试各个传输层的基本功能：

```bash
# 运行所有 MCP 单元测试
go test -short ./internal/mcp/

# 运行特定传输的测试
go test -short ./internal/mcp/ -run TestStdioTransport
go test -short ./internal/mcp/ -run TestHTTPTransport
go test -short ./internal/mcp/ -run TestSSETransport

# 查看覆盖率
go test -short -cover ./internal/mcp/
```

### 2. 集成测试

测试完整的 MCP 客户端功能：

```bash
# 运行集成测试（需要外部服务）
go test ./internal/mcp/ -run Integration -v
```

### 3. 错误恢复测试

测试自动重连和熔断机制：

```bash
# 运行错误恢复测试
go test ./internal/mcp/ -run TestCircuitBreaker -v
go test ./internal/mcp/ -run TestReconnect -v
```

## 功能测试清单

### Stdio Transport

- [ ] 启动和停止服务器进程
- [ ] 发送和接收 JSON-RPC 消息
- [ ] 处理进程崩溃
- [ ] 处理标准错误输出
- [ ] 环境变量传递
- [ ] 工作目录设置

### HTTP Transport

- [ ] HTTP POST 请求
- [ ] 自定义 Headers
- [ ] 超时处理
- [ ] 错误状态码处理
- [ ] 连接失败处理
- [ ] 请求重试

### SSE Transport

- [ ] SSE 连接建立
- [ ] 接收 SSE 事件
- [ ] 处理多行数据
- [ ] 连接超时
- [ ] 自动重连
- [ ] 自定义 Headers

### 通用功能

- [ ] JSON-RPC 协议
- [ ] 请求 ID 管理
- [ ] 并发请求处理
- [ ] Context 取消
- [ ] 超时控制
- [ ] 错误传播

## 高级测试场景

### 1. 并发测试

```bash
# 运行并发测试
go test ./internal/mcp/ -run Concurrent -v -race
```

### 2. 性能测试

```bash
# 运行基准测试
go test -bench=. ./internal/mcp/ -benchmem

# 查看内存分配
go test -bench=. ./internal/mcp/ -benchmem -memprofile mem.out
go tool pprof mem.out
```

### 3. 熔断器测试

```bash
# 测试熔断器状态转换
go test ./internal/mcp/ -run TestCircuitBreaker -v

# 测试场景：
# - Closed -> Open (失败次数达到阈值)
# - Open -> Half-Open (超时后)
# - Half-Open -> Closed (成功次数达到阈值)
# - Half-Open -> Open (再次失败)
```

### 4. 自动重连测试

```bash
# 测试自动重连机制
go test ./internal/mcp/ -run TestReconnect -v

# 测试场景：
# - 网络断开后自动重连
# - 重连失败后的退避策略
# - 最大重连次数
# - 重连成功后的状态恢复
```

## 实际 MCP 服务器测试

### 使用真实的 MCP 服务器

1. **安装 MCP 服务器**：

```bash
# 示例：安装文件系统 MCP 服务器
npm install -g @modelcontextprotocol/server-filesystem
```

2. **配置 GopherPaw**：

```yaml
mcp:
  servers:
    filesystem:
      name: "Filesystem MCP Server"
      description: "Access local filesystem"
      transport: "stdio"
      command: "mcp-server-filesystem"
      args: ["/path/to/allowed/directory"]
      enabled: true
```

3. **测试功能**：

```bash
# 启动 GopherPaw
./gopherpaw app

# 在对话中测试
> 请列出当前目录的文件
> 读取 test.txt 文件
> 创建一个新文件
```

## 测试工具和脚本

### MCP 服务器测试脚本

```bash
#!/bin/bash
# test_mcp_server.sh - 测试 MCP 服务器连接

SERVER_NAME="test-server"
CONFIG_FILE="configs/config.yaml"

# 1. 检查配置
if ! grep -q "$SERVER_NAME" "$CONFIG_FILE"; then
    echo "Error: MCP server not found in config"
    exit 1
fi

# 2. 启动测试
echo "Starting MCP server test..."

# 3. 发送测试请求
# (这里需要根据实际的 MCP 服务器实现)

# 4. 验证响应
# (检查日志或输出)

echo "MCP server test completed"
```

### 自动化测试脚本

```bash
#!/bin/bash
# run_all_mcp_tests.sh - 运行所有 MCP 测试

echo "Running MCP unit tests..."
go test -short ./internal/mcp/ -v

echo "Running MCP integration tests..."
go test ./internal/mcp/ -run Integration -v

echo "Running MCP benchmarks..."
go test -bench=. ./internal/mcp/ -benchmem

echo "All MCP tests completed"
```

## 常见问题

### 1. Stdio Transport 卡死

**原因**：管道缓冲区满或进程未正确响应

**解决方案**：
- 检查 MCP 服务器输出
- 确保服务器正确刷新输出（`flush=True`）
- 添加超时控制

### 2. HTTP Transport 连接失败

**原因**：网络问题或服务器未启动

**解决方案**：
- 检查服务器 URL
- 验证网络连接
- 检查防火墙设置
- 查看服务器日志

### 3. SSE Transport 事件丢失

**原因**：连接断开或事件格式错误

**解决方案**：
- 实现自动重连
- 验证 SSE 事件格式
- 添加心跳机制

### 4. 熔断器频繁打开

**原因**：MCP 服务器不稳定

**解决方案**：
- 调整熔断器阈值
- 检查服务器性能
- 增加重试次数
- 查看服务器日志

## 最佳实践

1. **先测试 Stdio** - 最简单，用于验证基本功能
2. **使用 Mock** - 单元测试使用 mock 服务器
3. **渐进式测试** - 从简单到复杂
4. **监控日志** - 设置 `log.level: debug`
5. **测试错误场景** - 网络断开、超时、错误响应
6. **性能基准** - 定期运行基准测试
7. **集成测试** - 在真实环境中测试

## 相关文档

- [MCP 协议规范](https://modelcontextprotocol.io/)
- [架构规范](./architecture_spec.md)
- [API 规范](./api_spec.md)
- [配置示例](../configs/config.yaml.example)
