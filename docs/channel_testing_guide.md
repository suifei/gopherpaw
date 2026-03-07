# GopherPaw 渠道测试指南

> 本文档提供 GopherPaw 各个消息渠道的测试方法和最佳实践

## 概述

GopherPaw 支持以下消息渠道：

- **Console** - 命令行交互（默认启用）
- **Telegram** - Telegram Bot
- **Discord** - Discord Bot
- **钉钉** - 钉钉机器人
- **飞书** - 飞书机器人
- **QQ** - QQ 机器人

## 测试类型

### 1. 单元测试

所有渠道都包含单元测试，测试基本功能：

```bash
# 运行所有渠道单元测试
go test -short ./internal/channels/

# 运行特定渠道的单元测试
go test -short ./internal/channels/ -run TestTelegramChannel
go test -short ./internal/channels/ -run TestDiscordChannel
go test -short ./internal/channels/ -run TestDingTalkChannel
go test -short ./internal/channels/ -run TestFeishuChannel
go test -short ./internal/channels/ -run TestQQChannel
```

### 2. 集成测试

集成测试使用 mock 对象模拟外部服务：

```bash
# 运行所有集成测试
go test ./internal/channels/ -run Integration

# 运行特定渠道的集成测试
go test ./internal/channels/ -run TestDingTalkIntegration
go test ./internal/channels/ -run TestDiscordIntegration
go test ./internal/channels/ -run TestFeishuIntegration
go test ./internal/channels/ -run TestQQIntegration
go test ./internal/channels/ -run TestTelegramIntegration
```

### 3. 端到端测试

端到端测试需要真实的 Bot Token 和配置。

## Console 渠道测试

Console 渠道是最简单的测试方式：

```bash
# 启动 Console 渠道
./gopherpaw app

# 或者使用配置文件
./gopherpaw app --config configs/config.yaml
```

## Telegram 渠道测试

### 1. 创建 Telegram Bot

1. 在 Telegram 中找到 `@BotFather`
2. 发送 `/newbot` 命令
3. 按提示设置 Bot 名称
4. 获取 Bot Token（格式：`1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`）

### 2. 配置环境变量

```bash
export GOPHERPAW_CHANNELS_TELEGRAM_BOT_TOKEN="your-bot-token-here"
```

### 3. 启动 Telegram 渠道

```bash
# 编辑配置文件
vim configs/config.yaml

# 启用 Telegram 渠道
channels:
  telegram:
    enabled: true
    bot_prefix: ""

# 启动
./gopherpaw app
```

### 4. 测试功能

1. 在 Telegram 中找到你的 Bot
2. 发送消息测试基本功能
3. 测试以下场景：
   - 发送文本消息
   - 发送图片
   - 发送文件
   - 长消息处理
   - Unicode 字符
   - 快速连续消息

## Discord 渠道测试

### 1. 创建 Discord Bot

1. 访问 [Discord Developer Portal](https://discord.com/developers/applications)
2. 创建新应用
3. 进入 Bot 页面，创建 Bot
4. 获取 Bot Token
5. 启用必要的 intents（Message Content Intent 等）

### 2. 邀请 Bot 到服务器

```
https://discord.com/api/oauth2/authorize?client_id=YOUR_CLIENT_ID&permissions=2048&scope=bot
```

### 3. 配置环境变量

```bash
export GOPHERPAW_CHANNELS_DISCORD_BOT_TOKEN="your-bot-token-here"
```

### 4. 启动 Discord 渠道

```yaml
channels:
  discord:
    enabled: true
    bot_prefix: "[BOT] "
```

```bash
./gopherpaw app
```

### 5. 测试功能

- 在 Discord 服务器中测试
- 测试文本消息、图片、文件
- 测试不同的频道
- 测试私聊功能

## 钉钉渠道测试

### 1. 创建钉钉机器人

1. 在钉钉群组中添加自定义机器人
2. 获取 Webhook URL
3. 配置安全设置（推荐使用签名验证）

### 2. 配置

```yaml
channels:
  dingtalk:
    enabled: true
    bot_prefix: "[BOT] "
    # 或使用环境变量
    # export GOPHERPAW_CHANNELS_DINGTALK_CLIENT_ID="your-client-id"
    # export GOPHERPAW_CHANNELS_DINGTALK_CLIENT_SECRET="your-client-secret"
```

### 3. 测试功能

- 测试发送消息到群组
- 测试 Webhook 接收消息
- 测试消息格式
- 测试并发消息

## 飞书渠道测试

### 1. 创建飞书应用

1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建企业自建应用
3. 获取 App ID 和 App Secret
4. 配置权限和事件订阅

### 2. 配置

```yaml
channels:
  feishu:
    enabled: true
    bot_prefix: "[BOT] "
    # 或使用环境变量
    # export GOPHERPAW_CHANNELS_FEISHU_APP_ID="your-app-id"
    # export GOPHERPAW_CHANNELS_FEISHU_APP_SECRET="your-app-secret"
```

### 3. 测试功能

- 测试发送消息
- 测试接收消息
- 测试富文本消息
- 测试文件发送

## QQ 渠道测试

### 1. 配置 QQ Bot

QQ 渠道需要使用 QQ 开放平台的 Bot：

1. 访问 QQ 开放平台
2. 创建 Bot 应用
3. 获取 App ID 和 Client Secret

### 2. 配置

```yaml
channels:
  qq:
    enabled: true
    bot_prefix: ""
    app_id: "your-app-id"
    client_secret: "your-client-secret"
```

### 3. 测试功能

- 测试发送消息
- 测试接收消息
- 测试群组消息
- 测试私聊

## 测试清单

### 基本功能测试

- [ ] 启动和停止渠道
- [ ] 发送文本消息
- [ ] 接收文本消息
- [ ] 处理长消息
- [ ] 处理 Unicode 字符
- [ ] 处理空消息

### 高级功能测试

- [ ] 发送图片
- [ ] 发送文件
- [ ] 接收图片
- [ ] 接收文件
- [ ] 消息编辑
- [ ] 消息删除

### 边界情况测试

- [ ] 网络断开重连
- [ ] 并发消息处理
- [ ] 消息队列满
- [ ] Context 取消
- [ ] 超时处理

### 性能测试

- [ ] 消息吞吐量
- [ ] 响应时间
- [ ] 内存使用
- [ ] 并发用户

## 测试工具

### 单元测试

```bash
# 运行所有单元测试
go test -short ./internal/channels/

# 运行特定测试
go test -short ./internal/channels/ -run TestConsoleChannelEdgeCases

# 查看覆盖率
go test -short -cover ./internal/channels/
```

### 基准测试

```bash
# 运行基准测试
go test -bench=. ./internal/channels/

# 运行特定基准测试
go test -bench=BenchmarkConsoleChannel ./internal/channels/
go test -bench=BenchmarkQueue ./internal/channels/
```

### 集成测试

```bash
# 运行集成测试（需要外部服务配置）
go test ./internal/channels/ -run Integration -v
```

## 常见问题

### 1. Bot Token 无效

- 检查 Token 格式
- 确认 Token 未过期
- 确认环境变量设置正确

### 2. 网络连接问题

- 检查网络连接
- 配置代理（如需要）
- 检查防火墙设置

### 3. 权限问题

- 确认 Bot 有必要的权限
- 检查频道/群组权限设置
- 确认用户有权限使用 Bot

### 4. 消息未送达

- 检查日志输出
- 确认 Webhook 配置正确
- 检查消息格式

## 最佳实践

1. **先测试 Console 渠道** - 确保核心功能正常
2. **使用环境变量** - 避免硬编码敏感信息
3. **启用日志** - 设置 `log.level: debug` 查看详细信息
4. **渐进式测试** - 从基本功能开始，逐步测试高级功能
5. **监控性能** - 使用基准测试监控性能变化
6. **错误处理** - 测试各种错误场景
7. **并发测试** - 测试多个用户同时使用的情况

## 相关文档

- [密钥管理最佳实践](../.cursor/skills/security_best_practices/SKILL.md)
- [配置文件示例](../configs/config.yaml.example)
- [架构规范](./architecture_spec.md)
- [API 规范](./api_spec.md)
