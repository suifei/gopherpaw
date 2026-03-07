#!/bin/bash
# GopherPaw Desktop 启动脚本
# 负责启动 VNC Server、XFCE 桌面、noVNC 代理和 GopherPaw 应用

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 清理函数
cleanup() {
    log_info "Cleaning up..."
    
    # 停止 GopherPaw
    if [ -f /tmp/gopherpaw.pid ]; then
        kill $(cat /tmp/gopherpaw.pid) 2>/dev/null || true
        rm -f /tmp/gopherpaw.pid
    fi
    
    # 停止 noVNC
    if [ -f /tmp/novnc.pid ]; then
        kill $(cat /tmp/novnc.pid) 2>/dev/null || true
        rm -f /tmp/novnc.pid
    fi
    
    # 停止 VNC Server
    vncserver -kill :1 2>/dev/null || true
    
    log_info "Cleanup completed"
}

# 注册清理钩子
trap cleanup EXIT

log_info "========================================="
log_info "GopherPaw Desktop Container Starting"
log_info "========================================="
log_info ""

# 检查环境变量
DISPLAY=${DISPLAY:-:1}
VNC_PASSWORD=${VNC_PASSWORD:-gopherpaw}
VNC_GEOMETRY=${VNC_GEOMETRY:-1920x1080}
VNC_DEPTH=${VNC_DEPTH:-24}
NOVNC_PORT=${NOVNC_PORT:-6080}
VNC_PORT=${VNC_PORT:-5901}
TZ=${TZ:-Asia/Shanghai}

log_info "Configuration:"
log_info "  Display: $DISPLAY"
log_info "  VNC Geometry: $VNC_GEOMETRY"
log_info "  VNC Depth: $VNC_DEPTH"
log_info "  noVNC Port: $NOVNC_PORT"
log_info "  VNC Port: $VNC_PORT"
log_info "  Timezone: $TZ"
log_info ""

# 1. 配置 VNC 密码
log_info "[1/5] Configuring VNC password..."

# 仅在密码不是默认值或密码文件不存在时重新生成
if [ "$VNC_PASSWORD" != "gopherpaw" ] || [ ! -f ~/.vnc/passwd ]; then
    mkdir -p ~/.vnc
    
    # 检查 vncpasswd 命令位置
    VNCPASSWD=""
    if [ -f /usr/bin/vncpasswd ]; then
        VNCPASSWD="/usr/bin/vncpasswd"
    elif [ -f /usr/bin/tigervncpasswd ]; then
        VNCPASSWD="/usr/bin/tigervncpasswd"
    else
        log_error "vncpasswd command not found!"
        exit 1
    fi
    
    # 创建密码文件
    echo "$VNC_PASSWORD" | $VNCPASSWD -f > ~/.vnc/passwd
    chmod 600 ~/.vnc/passwd
    log_info "VNC password configured using $VNCPASSWD"
else
    log_info "Using pre-configured VNC password"
fi
log_info ""

# 2. 启动 VNC Server
log_info "[2/5] Starting VNC Server..."

# 启动 VNC Server（使用密码认证）
vncserver $DISPLAY \
    -geometry $VNC_GEOMETRY \
    -depth $VNC_DEPTH \
    -rfbport $VNC_PORT \
    -localhost no \
    -PasswordFile ~/.vnc/passwd 2>&1

if [ $? -ne 0 ]; then
    log_error "Failed to start VNC server"
    exit 1
fi

log_info "VNC Server started on display $DISPLAY (port $VNC_PORT)"
log_info ""

# 3. 配置 XFCE 桌面会话
log_info "[3/5] Configuring XFCE Desktop session..."
export DISPLAY=$DISPLAY

# 配置 XFCE 会话（VNC 会自动执行 ~/.xsession）
cat > ~/.xsession <<'EOF'
#!/bin/bash
xrdb ~/.Xresources 2>/dev/null || true
startxfce4
EOF
chmod +x ~/.xsession

log_info "XFCE Desktop session configured"
log_info ""

# 4. 启动 noVNC 代理
log_info "[4/5] Starting noVNC WebSocket proxy..."

# 检查 noVNC 是否存在
if [ -d /usr/share/novnc ]; then
    NOVNC_DIR=/usr/share/novnc
elif [ -d /opt/novnc ]; then
    NOVNC_DIR=/opt/novnc
else
    log_error "noVNC not found"
    exit 1
fi

# 启动 noVNC
$NOVNC_DIR/utils/launch.sh \
    --vnc localhost:$VNC_PORT \
    --listen $NOVNC_PORT \
    > ~/.vnc/novnc.log 2>&1 &

NOVNC_PID=$!
echo $NOVNC_PID > /tmp/novnc.pid

sleep 2

if kill -0 $NOVNC_PID 2>/dev/null; then
    log_info "noVNC proxy started on port $NOVNC_PORT"
    log_info "Access URL: http://localhost:$NOVNC_PORT/vnc.html"
else
    log_error "Failed to start noVNC proxy"
    exit 1
fi

log_info ""

# 5. 启动 GopherPaw 应用
log_info "[5/5] Starting GopherPaw application..."

# 检查 gopherpaw 命令是否存在
if ! command -v gopherpaw &> /dev/null; then
    log_warn "GopherPaw binary not found, skipping..."
else
    # 在桌面环境中启动终端并运行 GopherPaw
    export DISPLAY=$DISPLAY
    
    # 方式 1: 在后台运行（无 GUI）
    # gopherpaw app start > /app/logs/gopherpaw.log 2>&1 &
    
    # 方式 2: 在 XFCE 终端中运行（推荐，用户可见）
    xfce4-terminal \
        --title="GopherPaw Agent" \
        --command="gopherpaw app start" \
        > /dev/null 2>&1 &
    
    GOPHERPAW_PID=$!
    echo $GOPHERPAW_PID > /tmp/gopherpaw.pid
    
    log_info "GopherPaw application started in XFCE terminal"
fi

log_info ""

# 完成
log_info "========================================="
log_info "Desktop Environment Ready!"
log_info "========================================="
log_info ""
log_info "Access Methods:"
log_info "  1. Web Browser: http://localhost:$NOVNC_PORT/vnc.html"
log_info "  2. VNC Client:   localhost:$VNC_PORT"
log_info "  3. Password:     $VNC_PASSWORD"
log_info ""
log_info "Press Ctrl+C to stop..."
log_info ""

# 保持脚本运行
tail -f /dev/null
