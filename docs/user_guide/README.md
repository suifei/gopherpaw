# GopherPaw 用户指南

## 快速开始

### 第一次运行

1. **启动 GopherPaw**

```bash
# 交互模式（默认）
./gopherpaw

# 新增：直接执行任务（非交互模式）
./gopherpaw app "搜索最新资讯" --once

# 新增：执行后继续交互
./gopherpaw app "帮我分析数据"
```

2. **开始对话**

```
你好！我是 GopherPaw，你的 AI 助手。
```

3. **使用内置命令**

输入 `/help` 查看所有可用命令。

## 规划-执行分离模式（v0.3.0 新增）

### 概述

GopherPaw v0.3.0 引入了规划-执行分离模式，支持复杂任务的自动规划和执行。

### 核心组件

- **TaskPlanner**: 任务规划器，根据用户请求生成结构化执行计划
- **Executor**: 执行器，按照计划执行具体步骤
- **ContextManager**: 上下文管理器，管理执行上下文和状态
- **CapabilityExtractor**: 能力提取器，从系统提取可用能力
- **SkillHook**: Skill 钩子，集成技能系统到执行流程
- **缓存系统**: 能力注册表和文件持久化

### 使用示例

```
用户: 用浏览器开同程的网站看看最新的旅游产品是什么，对比下携程，最后把选好的链接地址发给我

[TaskPlanner] 正在分析任务并生成执行计划...

执行计划：
1. 启动浏览器
2. 访问同程网站
3. 提取旅游产品信息
4. 访问携程网站
5. 提取旅游产品信息
6. 对比价格和服务
7. 生成推荐链接

[Executor] 开始执行...
[步骤 1/7] 正在启动浏览器...
[步骤 2/7] 正在访问同程网站...
...
```

### CLI 任务参数

```bash
# 单次执行模式
./gopherpaw app "帮我搜索最新资讯" --once

# 执行后继续交互
./gopherpaw app "帮我分析数据"

# 指定工作目录
./gopherpaw app "读取项目 README" --workdir /path/to/project
```

### 配置选项

```yaml
agent:
  planning:
    enabled: true          # 启用规划-执行模式
    cache_enabled: true    # 启用能力缓存
    max_steps: 10         # 最大执行步骤数
```

## 基本功能

### 1. 对话

直接输入消息即可与 AI 对话：

```
用户: 今天天气怎么样？
AI: 我无法直接查询天气，但可以帮你搜索最新的天气信息。
```

### 2. 使用工具

GopherPaw 会自动使用工具来完成任务：

```
用户: 帮我搜索 Go 语言教程
AI: [使用 web_search 工具]
我找到了一些优质的 Go 语言教程...
```

### 3. 文件操作

```
用户: 读取 /tmp/test.txt 文件
AI: [使用 read_file 工具]
文件内容是...
```

## 内置命令

### /help - 帮助信息

显示所有可用命令和使用说明。

### /compact - 压缩对话历史

当对话历史过长时，压缩旧的消息以节省 Token：

```
用户: /compact
AI: 已压缩对话历史。
```

### /clear - 清空上下文

清空当前会话的短期记忆：

```
用户: /clear
AI: 已清空上下文。
```

### /history - 查看对话历史

查看当前会话的所有消息：

```
用户: /history
AI: 
共 5 条消息。
  [1] user: 你好
  [2] assistant: 你好！有什么可以帮你的？
  [3] user: 今天天气
  [4] assistant: 我可以帮你查询天气...
  
预估 token 数: 150
```

### /new - 新建会话

保存当前会话到长期记忆并清空上下文：

```
用户: /new
AI: 已保存到长期记忆并清空上下文。
```

### /compact_str - 查看压缩摘要

查看对话历史的压缩摘要：

```
用户: /compact_str
AI: 
摘要内容：
- 讨论了 Go 语言基础
- 介绍了并发编程
- 推荐了几个学习资源
```

### /switch-model - 切换模型

切换 LLM 模型：

```
用户: /switch-model openai gpt-4o
AI: 已切换至 openai / gpt-4o
```

### /daemon - 守护进程管理

管理守护进程状态：

```
# 查看状态
/daemon status

# 查看版本
/daemon version

# 查看日志
/daemon logs 20

# 重新加载配置
/daemon reload-config

# 重启服务
/daemon restart
```

## 高级功能

### 1. 多轮对话

GopherPaw 会记住对话历史，支持多轮对话：

```
用户: 我在学习 Go
AI: 很好的选择！Go 是一门优秀的语言...

用户: 推荐一些学习资源
AI: 基于 Go 语言，我推荐以下资源...
```

### 2. 文件操作

#### 读取文件

```
用户: 读取 /path/to/file.txt
AI: [使用 read_file 工具]
文件内容：...
```

#### 写入文件

```
用户: 创建一个 test.txt 文件，内容是 "Hello World"
AI: [使用 write_file 工具]
已创建文件 /path/to/test.txt
```

#### 编辑文件

```
用户: 在 test.txt 中将 "Hello" 替换为 "Hi"
AI: [使用 edit_file 工具]
已替换 1 处内容
```

#### 追加内容

```
用户: 在 test.txt 末尾添加 "Goodbye"
AI: [使用 append_file 工具]
已追加内容
```

### 3. 执行命令

```
用户: 执行 ls -la 命令
AI: [使用 execute_shell_command 工具]
命令输出：
total 64
drwxr-xr-x  12 user user 4096 ...
```

### 4. 搜索功能

#### 网页搜索

```
用户: 搜索 Go 语言最佳实践
AI: [使用 web_search 工具]
搜索结果：
1. Effective Go - The Go Programming Language
2. Go Code Review Comments
...
```

#### 文件内容搜索

```
用户: 在 /path/to/dir 中搜索包含 "TODO" 的文件
AI: [使用 grep_search 工具]
找到 3 个匹配：
file1.go:15: // TODO: optimize this
file2.go:42: // TODO: add error handling
...
```

#### 文件名搜索

```
用户: 查找所有 .go 文件
AI: [使用 glob_search 工具]
找到 25 个文件：
main.go
utils.go
...
```

### 5. 记忆搜索

搜索历史对话：

```
用户: 搜索之前讨论过的并发相关内容
AI: [使用 memory_search 工具]
找到 3 条相关记忆：
1. [2024-01-15] 讨论了 goroutine 的使用
2. [2024-01-10] 介绍了 channel 的工作原理
...
```

## 使用技巧

### 1. 明确的指令

给出清晰、具体的指令：

```
❌ 差: 帮我看看这个文件
✅ 好: 读取 /path/to/file.txt 并总结主要内容
```

### 2. 分步执行

复杂任务分解为多个步骤：

```
用户: 
1. 搜索最新的 Go 1.22 特性
2. 整理成 Markdown 列表
3. 保存到 go122-features.md
```

### 3. 提供上下文

提供足够的背景信息：

```
用户: 我在使用 Go 1.22 开发一个 Web 服务，
需要实现 JWT 认证，推荐一个库并给出示例代码
```

### 4. 利用记忆

GopherPaw 会记住重要信息：

```
用户: 记住：我的项目使用 Go 1.22 和 Gin 框架
AI: 好的，我记住了你的项目技术栈。

[后续对话中]
用户: 给我写一个中间件
AI: 基于你的 Gin 框架，这是一个中间件示例...
```

## 常见场景

### 场景 1：代码审查

```
用户: 审查这段代码并指出问题：
[粘贴代码]

AI: 
代码审查结果：
1. 第 15 行：未处理错误
2. 第 23 行：可能的空指针引用
...
```

### 场景 2：文档生成

```
用户: 为 myapp.go 生成 API 文档

AI: [读取文件，分析代码]
# MyAPP API 文档

## 函数列表

### Func1
描述：...
参数：...
返回值：...
```

### 场景 3：项目脚手架

```
用户: 创建一个 Go Web 项目结构：
- cmd/server/main.go
- internal/handlers/
- internal/models/
- configs/config.yaml

AI: [创建目录和文件]
已创建项目结构：
...
```

### 场景 4：数据分析

```
用户: 分析 data.csv 文件，统计用户年龄分布

AI: [读取文件，执行分析]
年龄分布统计：
18-25岁: 35%
26-35岁: 45%
...
```

## 性能优化

### 1. 控制上下文长度

定期使用 `/compact` 或 `/clear` 清理上下文。

### 2. 选择合适的模型

```
/switch-model openai gpt-4o-mini  # 快速响应
/switch-model openai gpt-4o       # 复杂任务
```

### 3. 批量操作

```
用户: 一次性读取所有 .md 文件并生成目录
```

## 安全注意事项

### 1. 敏感信息

不要在对话中分享：
- API 密钥
- 密码
- 个人隐私信息

### 2. 文件访问

GopherPaw 只能访问配置允许的目录。

### 3. 命令执行

Shell 命令在沙盒环境中执行，但仍需谨慎。

## 故障排查

### 问题 1：工具执行失败

```
用户: 读取 /root/file.txt
AI: 错误：权限不足

解决方案：
1. 检查文件权限
2. 使用可访问的文件路径
```

### 问题 2：模型响应慢

```
解决方案：
1. 检查网络连接
2. 切换到更快的模型
3. 减少上下文长度
```

### 问题 3：记忆丢失

```
解决方案：
1. 使用 /history 检查记忆
2. 检查配置文件中的 memory 设置
3. 使用 /new 保存重要信息到长期记忆
```

## 快捷键

在终端中：
- `Ctrl+C`: 中断当前操作
- `Ctrl+D`: 退出程序
- `↑/↓`: 浏览历史命令

## 获取帮助

- **文档**: https://github.com/suifei/gopherpaw/docs
- **问题反馈**: https://github.com/suifei/gopherpaw/issues
- **社区讨论**: https://github.com/suifei/gopherpaw/discussions

## 更新日志

### v0.3.0
- 规划-执行分离模式
- CLI 任务参数支持
- HTML 内容自动提取
- 桌面容器化支持

### v1.0.0
- 核心功能发布
- 15+ 内置工具
- 多渠道支持
- 记忆系统
- MCP 客户端

---

**祝你使用愉快！** 🎉
