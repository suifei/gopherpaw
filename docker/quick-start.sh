#!/bin/bash
# GopherPaw Desktop 快速启动脚本（WSL 环境）

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}GopherPaw Desktop 快速启动${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker 未安装${NC}"
    echo "请先安装 Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

if ! docker ps &> /dev/null; then
    echo -e "${RED}错误: Docker 未运行${NC}"
    echo "请启动 Docker 服务"
    exit 1
fi

# 进入项目目录
cd /mnt/d/works/gateway/gopherpaw

# 1. 检查 .env 文件
if [ ! -f docker/.env ]; then
    echo -e "${YELLOW}[1/5] 创建 .env 配置文件...${NC}"
    cp docker/.env.template docker/.env 2>/dev/null || true
    
    echo -e "${YELLOW}请编辑 docker/.env 文件，设置以下配置：${NC}"
    echo "  - VNC_PASSWORD: 远程桌面密码"
    echo "  - GOPHERPAW_LLM_API_KEY: LLM API 密钥"
    echo ""
    read -p "配置完成后按回车继续..."
else
    echo -e "${GREEN}[1/5] .env 文件已存在${NC}"
fi

# 2. 构建 GopherPaw 二进制
echo -e "${YELLOW}[2/5] 构建 GopherPaw 二进制...${NC}"
go build -o gopherpaw ./cmd/gopherpaw/
echo -e "${GREEN}✓ 二进制构建完成${NC}"

# 3. 构建 Docker 镜像
echo -e "${YELLOW}[3/5] 构建 Docker 镜像（首次需要 10-20 分钟）...${NC}"
docker-compose -f docker/docker-compose.yml build

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Docker 镜像构建失败${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 镜像构建完成${NC}"

# 4. 启动容器
echo -e "${YELLOW}[4/5] 启动容器...${NC}"
docker-compose -f docker/docker-compose.yml up -d

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ 容器启动失败${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 容器已启动${NC}"

# 5. 等待服务就绪
echo -e "${YELLOW}[5/5] 等待服务就绪...${NC}"
sleep 5

# 检查容器状态
if docker-compose -f docker/docker-compose.yml ps | grep -q "Up"; then
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${GREEN}启动成功！${NC}"
    echo -e "${GREEN}=========================================${NC}"
    echo ""
    echo -e "${BLUE}访问方式：${NC}"
    echo "  浏览器: http://localhost:6080/vnc.html"
    echo "  密码: $(grep VNC_PASSWORD docker/.env | cut -d= -f2)"
    echo ""
    echo -e "${BLUE}常用命令：${NC}"
    echo "  查看日志: docker-compose -f docker/docker-compose.yml logs -f"
    echo "  停止容器: docker-compose -f docker/docker-compose.yml down"
    echo "  重启容器: docker-compose -f docker/docker-compose.yml restart"
    echo ""
    echo -e "${YELLOW}提示: 首次启动可能需要 10-30 秒初始化桌面环境${NC}"
else
    echo -e "${RED}容器启动异常，请检查日志${NC}"
    docker-compose -f docker/docker-compose.yml logs
    exit 1
fi
