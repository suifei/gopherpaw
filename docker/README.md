# GopherPaw Desktop 容器化部署指南

## 🚀 快速开始（5 分钟）

### 前置条件

- ✅ Docker 已安装并运行
- ✅ 至少 4GB 可用内存
- ✅ 至少 10GB 可用磁盘空间

### 一键启动

```bash
cd /mnt/d/works/gateway/gopherpaw
./docker/quick-start.sh
```

### 手动启动步骤

#### 1. 配置环境变量

```bash
cd /mnt/d/works/gateway/gopherpaw

# 复制配置模板
cp docker/.env.template docker/.env

# 编辑配置文件
vim docker/.env
```

**必须修改的配置：**

```bash
# VNC 密码（强密码）
VNC_PASSWORD=YourStrongPassword123!

# LLM API 密钥
GOPHERPAW_LLM_API_KEY=sk-your-api-key-here
```

#### 2. 构建 GopherPaw 二进制

```bash
go build -o gopherpaw ./cmd/gopherpaw/
```

#### 3. 构建 Docker 镜像（首次需要 10-20 分钟）

```bash
docker-compose -f docker/docker-compose.yml build
```

#### 4. 启动容器

```bash
docker-compose -f docker/docker-compose.yml up -d
```

#### 5. 访问桌面

浏览器打开：**http://localhost:6080/vnc.html**

输入密码：在 `docker/.env` 中配置的 `VNC_PASSWORD`

---

## 📋 配置说明

### 环境变量完整列表

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DOCKER_REGISTRY_MIRROR` | `docker.1ms.run` | Docker 镜像代理 |
| `VNC_PASSWORD` | `gopherpaw` | VNC 远程桌面密码 |
| `VNC_GEOMETRY` | `1920x1080` | 桌面分辨率 |
| `VNC_DEPTH` | `24` | 色深（24位真彩色） |
| `NOVNC_PORT` | `6080` | noVNC Web 端口 |
| `GOPHERPAW_LLM_API_KEY` | - | **必填**：LLM API 密钥 |
| `GOPHERPAW_LLM_BASE_URL` | `https://api.openai.com/v1` | LLM API 端点 |
| `GOPHERPAW_LOG_LEVEL` | `info` | 日志级别 |
| `TZ` | `Asia/Shanghai` | 时区 |

### 资源限制

默认配置：
- **CPU**: 1-2 核
- **内存**: 1-2 GB
- **临时文件**: 300 MB

如需调整，修改 `docker/docker-compose.yml` 中的 `deploy.resources` 部分。

---

## 🛠️ 常用命令

### 容器管理

```bash
# 查看容器状态
docker-compose -f docker/docker-compose.yml ps

# 查看日志（实时）
docker-compose -f docker/docker-compose.yml logs -f

# 查看特定服务日志
docker-compose -f docker/docker-compose.yml logs -f gopherpaw-desktop

# 进入容器
docker exec -it gopherpaw-desktop /bin/bash

# 停止容器
docker-compose -f docker/docker-compose.yml stop

# 启动容器
docker-compose -f docker/docker-compose.yml start

# 重启容器
docker-compose -f docker/docker-compose.yml restart

# 停止并删除容器
docker-compose -f docker/docker-compose.yml down

# 停止并删除容器+卷（危险）
docker-compose -f docker/docker-compose.yml down -v
```

### 调试命令

```bash
# 检查 VNC 进程
docker exec gopherpaw-desktop pgrep -a Xvnc

# 检查桌面环境
docker exec gopherpaw-desktop pgrep -a xfce

# 检查 noVNC 进程
docker exec gopherpaw-desktop pgrep -a websockify

# 查看 VNC 日志
docker exec gopherpaw-desktop cat ~/.vnc/*.log

# 测试 VNC 连接（需要 vncviewer）
vncviewer localhost:5901
```

---

## 🐛 故障排查

### 问题 1: 容器无法启动

**症状：** `docker-compose up -d` 失败

**检查步骤：**

```bash
# 1. 检查 Docker 服务
sudo systemctl status docker  # Linux
# 或在 Windows/Mac 上检查 Docker Desktop

# 2. 检查镜像是否构建成功
docker images | grep gopherpaw

# 3. 检查日志
docker-compose -f docker/docker-compose.yml logs
```

**解决方案：**
- 确保 Docker 正在运行
- 重新构建镜像：`docker-compose -f docker/docker-compose.yml build --no-cache`
- 检查资源是否足够（内存、磁盘）

---

### 问题 2: 无法访问 Web 界面

**症状：** 浏览器打开 `http://localhost:6080/vnc.html` 无响应

**检查步骤：**

```bash
# 1. 检查端口是否监听
netstat -an | grep 6080

# 2. 检查容器端口映射
docker port gopherpaw-desktop

# 3. 检查防火墙
sudo ufw status  # Ubuntu
# 或 Windows 防火墙设置
```

**解决方案：**
- 确认容器正在运行：`docker ps | grep gopherpaw`
- 检查端口冲突：修改 `docker/.env` 中的 `NOVNC_PORT`
- 禁用防火墙或添加例外规则

---

### 问题 3: VNC 密码错误

**症状：** 输入密码后提示"Authentication failed"

**解决方案：**

```bash
# 1. 确认密码配置
grep VNC_PASSWORD docker/.env

# 2. 重新设置密码
docker exec -it gopherpaw-desktop bash
echo "new_password" | vncpasswd -f > ~/.vnc/passwd
chmod 600 ~/.vnc/passwd
exit

# 3. 重启容器
docker-compose -f docker/docker-compose.yml restart
```

---

### 问题 4: 桌面环境卡顿

**症状：** 操作延迟高，画面卡顿

**解决方案：**

```bash
# 1. 降低分辨率（在 docker/.env 中）
VNC_GEOMETRY=1280x720

# 2. 增加资源限制（在 docker-compose.yml 中）
deploy:
  resources:
    limits:
      cpus: '3'
      memory: 3G

# 3. 优化 VNC 编码（在 Dockerfile 中）
ENV VNC_ENCODINGS=Tight
ENV VNC_COMPRESSION=6
ENV VNC_QUALITY=6
```

---

### 问题 5: GopherPaw 未启动

**症状：** 桌面环境中没有 GopherPaw 终端窗口

**检查步骤：**

```bash
# 1. 检查二进制文件
docker exec gopherpaw-desktop ls -lh /usr/local/bin/gopherpaw

# 2. 手动运行
docker exec -it gopherpaw-desktop bash
gopherpaw app start

# 3. 查看日志
docker exec gopherpaw-desktop cat /app/logs/gopherpaw.log
```

**解决方案：**
- 检查 `GOPHERPAW_LLM_API_KEY` 是否正确配置
- 查看错误日志：`docker-compose -f docker/docker-compose.yml logs`
- 确保配置文件存在：`docker exec gopherpaw-desktop ls /app/configs/`

---

## 🔐 安全建议

### 生产环境必做

1. **修改默认密码**
   ```bash
   # 使用强密码（至少 12 位，包含大小写、数字、特殊字符）
   VNC_PASSWORD=Str0ng!P@ssw0rd#2026
   ```

2. **启用 HTTPS**
   ```bash
   # 使用 Nginx 反向代理
   server {
       listen 443 ssl;
       server_name your-domain.com;
       
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
       
       location / {
           proxy_pass http://localhost:6080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

3. **限制访问**
   ```bash
   # 只允许特定 IP 访问（在 docker-compose.yml 中）
   networks:
     gopherpaw-network:
       driver: bridge
       ipam:
         config:
           - subnet: 172.28.0.0/16
   ```

4. **定期更新**
   ```bash
   # 定期重新构建镜像
   docker-compose -f docker/docker-compose.yml build --no-cache
   docker-compose -f docker/docker-compose.yml up -d
   ```

---

## 📊 性能监控

### 资源使用监控

```bash
# 实时监控
docker stats gopherpaw-desktop

# 查看详细信息
docker inspect gopherpaw-desktop | grep -A 10 "Memory"
```

### 日志分析

```bash
# 查看最近 100 行日志
docker-compose -f docker/docker-compose.yml logs --tail=100

# 导出日志到文件
docker-compose -f docker/docker-compose.yml logs > logs.txt
```

---

## 🎓 高级配置

### 自定义桌面环境

修改 `docker/Dockerfile.desktop`：

```dockerfile
# 安装额外软件
RUN apt-get install -y \
    firefox \
    code  # VS Code

# 配置 XFCE 主题
RUN mkdir -p ~/.themes && \
    wget https://example.com/theme.tar.gz && \
    tar -xzf theme.tar.gz -C ~/.themes
```

### 持久化数据

修改 `docker/docker-compose.yml`：

```yaml
services:
  gopherpaw-desktop:
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
```

### 多用户支持

```yaml
# docker-compose.multi-user.yml
version: '3.8'
services:
  gopherpaw-user1:
    extends:
      file: docker-compose.yml
      service: gopherpaw-desktop
    environment:
      - VNC_PASSWORD=user1_password
    ports:
      - "6081:6080"
  
  gopherpaw-user2:
    extends:
      file: docker-compose.yml
      service: gopherpaw-desktop
    environment:
      - VNC_PASSWORD=user2_password
    ports:
      - "6082:6080"
```

---

## 📞 获取帮助

- **文档**: `docs/DESKTOP_ROADMAP.md`
- **问题反馈**: https://github.com/suifei/gopherpaw/issues
- **配置示例**: `configs/config.yaml.example`

---

**最后更新**: 2026-03-07  
**版本**: v0.1.0-alpha
