---
name: security_best_practices
description: 密钥和敏感信息管理最佳实践指南
version: 1.0.0
author: GopherPaw Team
tags:
  - security
  - secrets
  - best-practices
---

# 密钥和敏感信息管理最佳实践

## 概述

本技能提供 GopherPaw 项目中密钥和敏感信息管理的最佳实践指南，帮助开发者避免密钥泄露风险。

## 核心原则

### 1. 永远不要提交密钥到代码仓库

**禁止行为**：
- ❌ 将 API 密钥硬编码在代码中
- ❌ 将 `config.yaml`（包含真实密钥）提交到 Git
- ❌ 在测试代码中使用真实密钥
- ❌ 在日志中输出密钥信息

**正确做法**：
- ✅ 使用环境变量管理密钥
- ✅ 使用 `.gitignore` 排除敏感文件
- ✅ 使用 pre-commit hooks 检查敏感信息
- ✅ 定期轮换密钥

## 环境变量管理

### 推荐的环境变量列表

```bash
# LLM 配置
export GOPHERPAW_LLM_API_KEY="your-api-key-here"
export GOPHERPAW_LLM_BASE_URL="https://api.openai.com/v1"
export GOPHERPAW_LLM_MODEL="gpt-4o-mini"

# Embedding 配置
export GOPHERPAW_EMBEDDING_API_KEY="your-embedding-key"
export GOPHERPAW_EMBEDDING_BASE_URL="https://api.openai.com/v1"
export GOPHERPAW_EMBEDDING_MODEL="text-embedding-3-small"

# Telegram Bot
export GOPHERPAW_CHANNELS_TELEGRAM_BOT_TOKEN="your-bot-token"

# Discord Bot
export GOPHERPAW_CHANNELS_DISCORD_BOT_TOKEN="your-bot-token"

# 钉钉
export GOPHERPAW_CHANNELS_DINGTALK_CLIENT_ID="your-client-id"
export GOPHERPAW_CHANNELS_DINGTALK_CLIENT_SECRET="your-client-secret"

# 飞书
export GOPHERPAW_CHANNELS_FEISHU_APP_ID="your-app-id"
export GOPHERPAW_CHANNELS_FEISHU_APP_SECRET="your-app-secret"

# 工作目录
export GOPHERPAW_WORKING_DIR="~/.gopherpaw"
export GOPHERPAW_SECRET_DIR="~/.gopherpaw.secret"
```

### 环境变量设置方法

**Linux/macOS**：
```bash
# 方法1: 在 ~/.bashrc 或 ~/.zshrc 中设置
echo 'export GOPHERPAW_LLM_API_KEY="your-key"' >> ~/.bashrc
source ~/.bashrc

# 方法2: 使用 .env 文件（不提交到 Git）
# 创建 .env 文件
cat > .env << 'EOF'
export GOPHERPAW_LLM_API_KEY="your-key"
export GOPHERPAW_LLM_BASE_URL="https://api.openai.com/v1"
EOF

# 加载 .env 文件
source .env
```

**Windows**：
```powershell
# 临时设置（当前会话）
$env:GOPHERPAW_LLM_API_KEY="your-key"

# 永久设置（用户级别）
[Environment]::SetEnvironmentVariable("GOPHERPAW_LLM_API_KEY", "your-key", "User")
```

**Docker**：
```bash
# docker run 时传入
docker run -e GOPHERPAW_LLM_API_KEY="your-key" gopherpaw

# 或使用 --env-file
docker run --env-file .env gopherpaw
```

## 配置文件管理

### .gitignore 配置

确保以下文件被忽略：

```gitignore
# 配置文件（包含密钥）
config.yaml
configs/config.yaml

# 环境变量文件
.env
.env.local
.env.*.local

# 密钥目录
.secret/
*.secret

# 凭证文件
credentials.json
secrets.json
envs.json
providers.json
```

### config.yaml.example 模板

只包含占位符和示例：

```yaml
llm:
  provider: openai
  api_key: ""  # 使用环境变量 GOPHERPAW_LLM_API_KEY
  base_url: ""  # 使用环境变量 GOPHERPAW_LLM_BASE_URL
```

## Pre-commit Hook 检查

项目已配置 pre-commit hook，会在每次提交前自动检查敏感信息。

### 查看和测试 Hook

```bash
# Hook 位置
cat .git/hooks/pre-commit

# 手动测试（模拟提交）
git add .
git commit -m "test"  # 如果发现敏感信息会被阻止
```

### 跳过检查（不推荐）

```bash
git commit --no-verify
```

## 密钥轮换流程

### 定期轮换（推荐每 90 天）

1. **生成新密钥**：
   - 在服务商控制台创建新密钥
   - 不要立即删除旧密钥

2. **更新环境变量**：
   ```bash
   export GOPHERPAW_LLM_API_KEY="new-api-key"
   ```

3. **测试新密钥**：
   ```bash
   go run ./cmd/gopherpaw/ test
   ```

4. **删除旧密钥**：
   - 确认新密钥正常工作后，在服务商控制台删除旧密钥

### 泄露后的紧急轮换

1. **立即撤销泄露的密钥**
2. **生成新密钥并更新所有环境**
3. **检查日志，评估影响范围**
4. **审查 Git 历史，必要时使用 BFG Repo-Cleaner 清理**

## 检查清单

### 提交前检查

- [ ] 确认 `config.yaml` 未包含在提交中
- [ ] 运行 pre-commit hook 检查
- [ ] 确认测试代码使用的是占位符而非真实密钥
- [ ] 检查日志输出不包含敏感信息

### 定期审查（每月）

- [ ] 检查 `.gitignore` 是否完整
- [ ] 审查环境变量是否安全存储
- [ ] 检查密钥是否需要轮换
- [ ] 审查团队成员的访问权限

## 工具和资源

### 密钥检测工具

- **git-secrets**: https://github.com/awslabs/git-secrets
- **truffleHog**: https://github.com/trufflesecurity/trufflehog
- **gitleaks**: https://github.com/gitleaks/gitleaks

### 密钥管理服务

- **HashiCorp Vault**: 企业级密钥管理
- **AWS Secrets Manager**: AWS 环境推荐
- **Azure Key Vault**: Azure 环境推荐
- **1Password CLI**: 小团队推荐

## 相关文档

- [GopherPaw 配置文档](../../docs/api_spec.md#环境变量覆盖)
- [OWASP 密钥管理指南](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheatsheet.html)

## 最佳实践总结

1. ✅ **环境变量优先**：始终使用环境变量管理密钥
2. ✅ **最小权限原则**：只授予必要的权限
3. ✅ **定期轮换**：每 90 天轮换一次密钥
4. ✅ **多层防护**：使用 .gitignore + pre-commit hooks + 密钥检测工具
5. ✅ **快速响应**：发现泄露后立即撤销密钥
6. ✅ **审计日志**：记录密钥的使用和变更
7. ✅ **团队培训**：定期培训团队成员的安全意识

## 帮助和支持

如有疑问，请联系：
- 项目维护者：[GitHub Issues](https://github.com/suifei/gopherpaw/issues)
- 安全问题：请通过私密渠道报告
