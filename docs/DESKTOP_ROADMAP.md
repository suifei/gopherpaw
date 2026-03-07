# GopherPaw 桌面容器平台 - 实施路线图

## 📊 当前进度

### ✅ 已完成（Phase 1: App 管理模块）

- [x] **[1/6]** 定义接口 - `internal/app/types.go`
  - App 核心结构
  - Manager 接口
  - 生命周期钩子
  - 重启选项

- [x] **[2/6]** 实现启动序列 - `internal/app/app.go` (MVP 版本)
  - NewApp() 构造函数
  - Start() 启动流程
  - Stop() 停止流程（逆序）
  - HealthCheck() 健康检查

- [x] **[3/6]** 编写测试 - `internal/app/app_test.go`
  - 6 个测试用例全部通过
  - 覆盖正常流程和边界情况

- [x] **[4/6]** 集成到 CLI（部分）
  - 已识别现有 `cmd/gopherpaw/app.go`
  - 需要修改现有代码而不是创建新文件

### ✅ 已完成（Phase 3: Docker 容器化）

- [x] **Dockerfile.desktop** - 完整的容器镜像
  - Ubuntu 22.04 + XFCE + TigerVNC + noVNC
  - 构建时预生成 VNC 密码
  - 健康检查（X11 socket 检测）
  - 镜像大小：2.13GB

- [x] **start-desktop.sh** - 优化的启动脚本
  - VNC Server 启动
  - XFCE 桌面会话配置
  - noVNC 代理启动
  - GopherPaw 应用启动

- [x] **docker-compose.yml** - 生产就绪配置
  - 端口映射（6080/5901/8081）
  - 资源限制（2GB 内存）
  - 健康检查配置
  - 自动重启策略

- [x] **验证通过**
  - 容器状态：healthy
  - noVNC Web 访问正常
  - GopherPaw 可运行
  - 所有依赖工具已安装

- [x] **优化完成**
  - 删除冗余补丁脚本（fix-vnc.sh）
  - 修复健康检查机制
  - 优化 VNC 密码管理
  - 优化 XFCE 启动方式

### ⏸️ 进行中

- [ ] **更新契约文档**
  - `docs/architecture_spec.md` - 需要新增 app 模块说明
  - `docs/api_spec.md` - 需要新增接口定义

- [ ] **验证 GopherPaw 功能**
  - 在 XFCE 终端中测试工具调用
  - 配置 LLM API 密钥
  - 运行简单的 Agent 任务

---

## 🎯 后续实施计划

### Phase 2: Desktop 桌面管理模块（3-4 天）

```
优先级：🔴 P0 (核心需求)
```

#### 任务清单

- [ ] **创建基础类型** - `internal/desktop/types.go`
  ```go
  type Manager struct {
      VNCServer      *VNCServer
      NoVNCProxy     *NoVNCProxy
      SessionRecorder *SessionRecorder
      ControlSwitcher *ControlSwitcher
  }
  
  type VNCServer struct {
      Display  string
      Password string
      Process  *os.Process
  }
  ```

- [ ] **实现 VNC 管理** - `internal/desktop/vnc_server.go`
  - `Start()` - 启动 TigerVNC 服务器
  - `Stop()` - 停止 VNC 服务器
  - `IsRunning()` - 检查运行状态

- [ ] **实现 noVNC 代理** - `internal/desktop/novnc_proxy.go`
  - `Start()` - 启动 WebSocket 代理
  - `Stop()` - 停止代理
  - `GetURL()` - 获取访问 URL

- [ ] **编写测试** - `internal/desktop/*_test.go`
  - VNC 启动/停止测试
  - noVNC 代理测试
  - 端口冲突测试

- [ ] **集成到 App** - 修改 `internal/app/app.go`
  ```go
  func (a *App) startDesktop(ctx context.Context) error {
      desktopMgr := desktop.NewManager(a.Config.Desktop)
      if err := desktopMgr.Start(ctx); err != nil {
          return err
      }
      a.DesktopManager = desktopMgr
      return nil
  }
  ```

---

### Phase 3: Docker 容器化 ✅

```
优先级：🔴 P0 (核心需求)
状态：✅ 已完成
完成日期：2026-03-08
```

#### 已完成任务

- [x] **创建 Dockerfile** - `docker/Dockerfile.desktop`
  - ✅ Ubuntu 22.04 基础镜像
  - ✅ XFCE + TigerVNC + noVNC 集成
  - ✅ 构建时预生成 VNC 密码
  - ✅ 健康检查配置（X11 socket 检测）
  
- [x] **创建启动脚本** - `docker/scripts/start-desktop.sh`
  - ✅ VNC Server 启动（Xtigervnc）
  - ✅ XFCE 桌面会话配置（~/.xsession）
  - ✅ noVNC 代理启动
  - ✅ GopherPaw 应用启动
  - ✅ 优化的密码管理（支持环境变量覆盖）

- [x] **创建依赖安装脚本** - `docker/scripts/install-dependencies.sh`
  - ✅ LibreOffice (soffice)
  - ✅ Poppler (pdftoppm)
  - ✅ Pandoc
  - ✅ FFmpeg
  - ✅ Git

- [x] **创建 Docker Compose** - `docker/docker-compose.yml`
  - ✅ 端口映射：6080 (noVNC), 5901 (VNC), 8081 (GopherPaw)
  - ✅ 环境变量配置
  - ✅ 资源限制（2GB 内存）
  - ✅ 健康检查（X11 socket 检测）
  - ✅ 自动重启策略

- [x] **端到端验证**
  - ✅ 镜像构建成功（2.13GB）
  - ✅ 容器启动成功
  - ✅ VNC/noVNC 可访问（http://localhost:6080/vnc.html）
  - ✅ 健康检查通过（healthy）
  - ✅ GopherPaw 可运行
  - ✅ 所有依赖工具已安装

#### 优化记录

1. **健康检查修复**
   - 原问题：`pgrep -x Xvnc` 检测不到 `Xtigervnc` 进程
   - 解决方案：改用 `test -S /tmp/.X11-unix/X1` 检测 X11 socket
   - 文件：`docker/Dockerfile.desktop:110`, `docker/docker-compose.yml:73`

2. **VNC 密码优化**
   - 原问题：每次启动都重新生成密码
   - 解决方案：构建时预生成默认密码，支持环境变量覆盖
   - 文件：`docker/Dockerfile.desktop:78-81`, `docker/scripts/start-desktop.sh:74-93`

3. **XFCE 启动优化**
   - 原问题：XFCE 在后台启动导致 VNC session 退出
   - 解决方案：使用 VNC 标准的 ~/.xsession 配置
   - 文件：`docker/scripts/start-desktop.sh:120-133`

4. **删除冗余补丁**
   - 删除文件：`docker/fix-vnc.sh`
   - 原因：功能已集成到 Dockerfile 和启动脚本

---

### Phase 4: 控制切换与录制（可选，2-3 天）

```
优先级：🟡 P1 (增强功能)
```

#### 任务清单

- [ ] **实现控制切换器** - `internal/desktop/control_switcher.go`
  - Agent 模式（自动执行）
  - User 模式（用户控制）
  - Cooperative 模式（协作）

- [ ] **实现会话录制** - `internal/desktop/session_recorder.go`
  - 录制桌面事件
  - 导出为视频/JSON
  - 回放功能

- [ ] **WebSocket 集成** - `internal/app/server.go`
  - 广播桌面事件
  - 控制切换通知

---

### Phase 5: 文档与回填（1 天）

```
优先级：🟡 P1 (必须完成)
```

#### 任务清单

- [ ] **更新架构文档** - `docs/architecture_spec.md`
  ```markdown
  ## internal/app/ 模块
  
  职责：统一应用生命周期管理
  
  依赖关系：
  - agent/ (核心引擎)
  - channels/ (消息渠道)
  - scheduler/ (定时任务)
  - mcp/ (MCP 集成)
  - desktop/ (远程桌面)
  ```

- [ ] **更新接口文档** - `docs/api_spec.md`
  ```markdown
  ## App 接口
  
  ```go
  type Manager interface {
      Start(ctx context.Context) error
      Stop(ctx context.Context) error
      RestartServices(ctx context.Context, opts RestartOptions) error
      HealthCheck() map[string]bool
  }
  ```
  ```

- [ ] **创建用户指南** - `docs/desktop_platform_guide.md`
  - 快速开始
  - 配置说明
  - 故障排查
  - 性能优化

- [ ] **更新 CONTEXT.md**
  - 当前状态
  - 更新日志
  - 下一步计划

---

## 🛠️ 关键技术决策记录

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 桌面环境 | **XFCE** | 性能与功能平衡，资源消耗 ~600MB |
| 远程协议 | **noVNC + TigerVNC** | Web 集成友好，浏览器直接访问 |
| VNC 编码 | **Tight** | 带宽消耗低，延迟 <100ms |
| 用户模式 | **单容器单用户** | 隔离性好，资源限制明确 |
| App 管理 | **新增 internal/app/** | 参考 CoPaw，统一生命周期管理 |
| 依赖安装 | **自动安装 + 手动回退** | detector.go 检测 + installer.go 安装 |
| 热重载 | **Single-Flight + 原子切换** | 参考 CoPaw _restart_services() |

---

## 📈 资源消耗估算

### 单容器配置

| 资源 | 最小配置 | 推荐配置 |
|------|---------|---------|
| CPU | 1 核 | 2 核 |
| 内存 | 1.5 GB | 2 GB |
| 磁盘 | 8 GB | 15 GB |
| 网络 | 1 Mbps | 5 Mbps |

### 多用户场景（10 用户）

| 方案 | 总内存 | 总 CPU | 推荐配置 |
|------|--------|--------|---------|
| 单容器单用户 | 6-10 GB | 80-200% | 4 核 16GB |
| 共享容器 | 2-4 GB | 40-100% | 2 核 8GB |

---

## 🚀 快速启动指南（MVP）

### 1. 配置环境变量

```bash
cd /mnt/d/works/gateway/gopherpaw
cp docker/.env.template docker/.env

# 编辑 .env 文件
vim docker/.env
```

```bash
# docker/.env 内容
DOCKER_REGISTRY_MIRROR=docker.1ms.run
VNC_PASSWORD=your_secure_password_here
GOPHERPAW_LLM_API_KEY=sk-your-api-key
GOPHERPAW_LLM_BASE_URL=https://api.openai.com/v1
```

### 2. 构建并启动容器

```bash
# 构建镜像（首次需要 10-20 分钟）
docker-compose -f docker/docker-compose.yml build

# 启动容器
docker-compose -f docker/docker-compose.yml up -d

# 查看日志
docker-compose -f docker/docker-compose.yml logs -f
```

### 3. 访问桌面

```
浏览器访问: http://localhost:6080/vnc.html
密码: your_secure_password_here (在 .env 中配置)
```

### 4. 停止容器

```bash
docker-compose -f docker/docker-compose.yml down
```

---

## 🎓 学习资源

### CoPaw 参考实现

- **应用生命周期**: `copaw-source/src/copaw/app/_app.py`
  - lifespan() 启动序列
  - _restart_services() 热重载机制

- **Channel Manager**: `copaw-source/src/copaw/app/channels/manager.py`
  - 多渠道管理
  - 热重载支持

- **Cron Manager**: `copaw-source/src/copaw/app/crons/manager.py`
  - APScheduler 集成

### 远程桌面技术

- **TigerVNC**: https://github.com/TigerVNC/tigervnc
- **noVNC**: https://github.com/novnc/noVNC
- **XFCE 优化**: https://xfce.org/wiki

---

## 🐛 已知问题与解决方案

### 问题 1: VNC 延迟过高（>200ms）

**解决方案:**
```bash
# 在 Dockerfile 中配置 Tight 编码
ENV VNC_ENCODINGS=Tight
ENV VNC_COMPRESSION=6
ENV VNC_QUALITY=6
```

### 问题 2: 容器内存占用过大（>2GB）

**解决方案:**
```yaml
# docker-compose.yml 中添加资源限制
deploy:
  resources:
    limits:
      memory: 2G
    reservations:
      memory: 1.5G
```

### 问题 3: 热重载导致服务中断

**解决方案:**
```go
// 参考 CoPaw 实现 Single-Flight 重启
if a.restartTask != nil && !a.restartTask.done() {
    await a.restartTask  // 等待正在进行的重启
    return
}
```

---

## 📝 下一步行动（优先级排序）

### 🔴 紧急（本周内完成）

1. ✅ **完成 Phase 3: Docker 容器化** - 已完成 (2026-03-08)
   - ✅ 创建 `docker/Dockerfile.desktop`
   - ✅ 创建 `docker/scripts/start-desktop.sh`
   - ✅ 创建 `docker/docker-compose.yml`
   - ✅ 端到端验证
   - ✅ 移除 fix-vnc.sh 补丁脚本
   - ✅ 优化健康检查机制
   - ✅ 优化 VNC 密码管理

2. **验证 GopherPaw 功能（进行中）**
   - [ ] 在 XFCE 终端中运行 `gopherpaw --help`
   - [ ] 测试文件操作工具
   - [ ] 测试浏览器工具
   - [ ] 配置 LLM API 密钥
   - [ ] 运行简单的 Agent 任务

3. **更新契约文档**
   - [ ] `docs/architecture_spec.md` - 新增 desktop 模块
   - [ ] `docs/api_spec.md` - 新增 Desktop Manager 接口

### 🟡 重要（下周完成）

4. **完成 Phase 2: Desktop 模块**
   - [ ] 实现 VNC 管理（Go 代码）
   - [ ] 实现 noVNC 代理管理
   - [ ] 控制切换功能
   - [ ] 会话录制
   - [ ] 集成测试

5. **完善 CLI 集成**
   - [ ] 修改 `cmd/gopherpaw/app.go`
   - [ ] 添加 `desktop` 子命令
   - [ ] 添加状态查询命令

### 🟢 次要（后续迭代）

6. **实现 Phase 4: 控制切换**
   - [ ] Agent/User 模式切换
   - [ ] 会话录制与回放
   - [ ] WebSocket 事件广播

7. **生产环境优化**
   - [ ] HTTPS 配置
   - [ ] 多用户支持
   - [ ] VNC 编码优化
   - [ ] 内存占用优化
   - [ ] 启动速度优化

---

## 📊 里程碑时间表

```
Week 1 (当前):
  ├─ Day 1-2: Phase 3 Docker 容器化 ✅ (已完成 - 2026-03-08)
  ├─ Day 3: GopherPaw 功能验证 + Desktop 模块基础
  └─ Day 4-5: 集成测试 + 文档更新

Week 2:
  ├─ Day 1-2: Desktop 完整实现
  ├─ Day 3: CLI 完善
  ├─ Day 4: Phase 4 控制切换
  └─ Day 5: 性能优化

Week 3:
  ├─ Day 1-2: 生产环境测试
  ├─ Day 3-4: 文档完善
  └─ Day 5: 发布准备
```

---

## 🤝 贡献指南

### 开发环境设置

```bash
# 1. 克隆仓库
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw

# 2. 安装依赖
go mod download

# 3. 运行测试
go test ./...

# 4. 构建二进制
go build -o gopherpaw ./cmd/gopherpaw/
```

### 代码规范

- 遵循 `gofmt` 格式化
- 使用 `go vet` 静态检查
- 所有导出函数必须有 godoc 注释
- 测试覆盖率 > 80%

---

## 📞 支持

- **问题反馈**: https://github.com/suifei/gopherpaw/issues
- **文档**: `docs/` 目录
- **示例配置**: `configs/config.yaml.example`

---

**最后更新**: 2026-03-08
**版本**: v0.1.0-alpha (MVP)
**维护者**: GopherPaw Team
