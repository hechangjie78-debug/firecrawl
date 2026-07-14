#!/usr/bin/env bash
# ============================================================
# Firecrawl 一键部署脚本（混合架构）
# 中间件(PostgreSQL/Redis/RabbitMQ) → Docker 容器
# 应用(Firecrawl API/Workers/Playwright) → 裸机 systemd
# ============================================================
set -euo pipefail

# ---------- 配置（按需修改） ----------
REPO_URL="https://github.com/hechangjie78-debug/firecrawl.git"
BRANCH="main"
INSTALL_DIR="/opt/firecrawl"
NODE_MAJOR=22
GO_VERSION="1.23.8"
NUQ_WORKER_COUNT=5
PLAYWRIGHT_PORT=3003
API_PORT=3002

PG_USER=firecrawl
PG_PASS=firecrawl
PG_DB=firecrawl

# ---------- 颜色 ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }
ok()    { echo -e "${CYAN}[OK]${NC}    $1"; }

# ---------- 前置检查 ----------
precheck() {
    if [[ $EUID -ne 0 ]]; then
        error "请以 root 用户运行（sudo ./deploy.sh）"
        exit 1
    fi
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
            error "当前仅支持 Ubuntu/Debian，检测到 $ID"
            exit 1
        fi
    fi
    info "前置检查通过"
}

# ========== 1. 系统依赖 ==========
install_system_deps() {
    info "安装系统依赖..."
    apt-get update -qq
    apt-get install -y -qq \
        build-essential pkg-config curl git python3 gnupg \
        ca-certificates lsb-release unzip jq sudo procps \
        libnss3 libnspr4 libatk1.0-0t64 libatk-bridge2.0-0t64 \
        libcups2t64 libdrm2 libdbus-1-3 libxkbcommon0 \
        libxcomposite1 libxdamage1 libxrandr2 libgbm1 \
        libpango-1.0-0 libcairo2 libasound2t64 libatspi2.0-0t64 \
        libwayland-client0 libwayland-egl1
    info "系统依赖安装完成"
}

# ========== 2. Node.js ==========
install_nodejs() {
    if command -v node &>/dev/null; then
        local ver=$(node -v | sed 's/v//' | cut -d. -f1)
        if [[ $ver -ge $NODE_MAJOR ]]; then
            info "Node.js $(node -v) 已安装，跳过"
            return
        fi
        warn "Node.js 版本过低 ($(node -v))，将升级"
    fi
    info "安装 Node.js $NODE_MAJOR ..."
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key \
        -o /etc/apt/keyrings/nodesource.asc
    chmod 0644 /etc/apt/keyrings/nodesource.asc
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.asc] https://deb.nodesource.com/node_${NODE_MAJOR}.x nodistro main" \
        > /etc/apt/sources.list.d/nodesource.list
    apt-get update -qq && apt-get install -y -qq nodejs
    info "Node.js $(node -v) 安装完成"
}

# ========== 3. pnpm ==========
install_pnpm() {
    if command -v pnpm &>/dev/null; then
        info "pnpm $(pnpm -v) 已安装，跳过"
        return
    fi
    npm install -g pnpm
    info "pnpm $(pnpm -v) 安装完成"
}

# ========== 4. Go ==========
install_go() {
    if command -v go &>/dev/null && go version | grep -q "go${GO_VERSION%.*}"; then
        info "Go $(go version) 已安装，跳过"
        return
    fi
    info "安装 Go $GO_VERSION ..."
    local tarball="go${GO_VERSION}.linux-amd64.tar.gz"
    curl -fsSL "https://go.dev/dl/${tarball}" -o "/tmp/${tarball}"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${tarball}"
    ln -sf /usr/local/go/bin/go /usr/local/bin/go
    rm "/tmp/${tarball}"
    info "Go $(go version) 安装完成"
}

# ========== 5. Docker ==========
install_docker() {
    if command -v docker &>/dev/null; then
        info "Docker $(docker --version) 已安装，跳过"
        return
    fi
    info "安装 Docker ..."
    curl -fsSL https://get.docker.com | bash
    systemctl enable docker --now
    info "Docker $(docker --version) 安装完成"
}

# ========== 6. Docker Compose（inline，中间件） ==========
create_compose_file() {
    local compose_dir="$INSTALL_DIR/deploy"
    mkdir -p "$compose_dir"

    cat > "$compose_dir/docker-compose-infra.yml" <<EOF
version: "3.9"
name: firecrawl-infra
services:
  postgres:
    image: postgres:17
    restart: unless-stopped
    ports:
      - "127.0.0.1:5432:5432"
    environment:
      POSTGRES_USER: ${PG_USER}
      POSTGRES_PASSWORD: ${PG_PASS}
      POSTGRES_DB: ${PG_DB}
    volumes:
      - firecrawl-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${PG_USER} -d ${PG_DB}"]
      interval: 5s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - firecrawl-redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 10

  rabbitmq:
    image: rabbitmq:3-management
    restart: unless-stopped
    ports:
      - "127.0.0.1:5672:5672"
      - "127.0.0.1:15672:15672"
    volumes:
      - firecrawl-rabbitmqdata:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 5s
      timeout: 5s
      retries: 10

volumes:
  firecrawl-pgdata:
  firecrawl-redisdata:
  firecrawl-rabbitmqdata:
EOF
    ok "docker-compose-infra.yml 已生成"
}

# ========== 启动中间件容器 ==========
start_middleware() {
    info "启动中间件容器（PostgreSQL / Redis / RabbitMQ）..."

    local compose_dir="$INSTALL_DIR/deploy"
    cd "$compose_dir"

    # 拉取最新镜像
    docker compose -f docker-compose-infra.yml pull -q

    # 启动（幂等）
    docker compose -f docker-compose-infra.yml up -d --remove-orphans

    # 等待 PostgreSQL 就绪
    info "等待 PostgreSQL 就绪..."
    for i in $(seq 1 30); do
        if docker compose -f docker-compose-infra.yml exec -T postgres \
            pg_isready -U "$PG_USER" -d "$PG_DB" &>/dev/null; then
            ok "PostgreSQL 就绪"
            break
        fi
        sleep 2
    done

    # 创建 pgcrypto 扩展
    docker compose -f docker-compose-infra.yml exec -T postgres \
        psql -U "$PG_USER" -d "$PG_DB" -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;" 2>/dev/null || true

    # 等待 Redis 就绪
    info "等待 Redis 就绪..."
    for i in $(seq 1 15); do
        if docker compose -f docker-compose-infra.yml exec -T redis \
            redis-cli ping 2>/dev/null | grep -q PONG; then
            ok "Redis 就绪"
            break
        fi
        sleep 2
    done

    # 等待 RabbitMQ 就绪
    info "等待 RabbitMQ 就绪..."
    for i in $(seq 1 30); do
        if docker compose -f docker-compose-infra.yml exec -T rabbitmq \
            rabbitmq-diagnostics ping 2>/dev/null | grep -q "reply.*ok"; then
            ok "RabbitMQ 就绪"
            break
        fi
        sleep 2
    done

    cd "$INSTALL_DIR"
}

# ========== 克隆 / 拉取 ==========
clone_or_pull() {
    if [[ -d "$INSTALL_DIR/.git" ]]; then
        info "仓库已存在，拉取最新代码..."
        cd "$INSTALL_DIR"
        git fetch origin "$BRANCH"
        git reset --hard "origin/$BRANCH"
    else
        info "克隆仓库 $REPO_URL ..."
        mkdir -p "$(dirname "$INSTALL_DIR")"
        git clone --branch "$BRANCH" --depth 1 "$REPO_URL" "$INSTALL_DIR"
    fi
    cd "$INSTALL_DIR"
    info "当前 commit: $(git log --oneline -1)"
}

# ========== 构建 ==========
build_all() {
    info "开始构建..."

    # 1. pnpm install
    cd "$INSTALL_DIR"
    if ls apps/*/package.json &>/dev/null; then
        pnpm install --frozen-lockfile 2>/dev/null || pnpm install
    fi

    # 2. Go 共享库 (libhtml-to-markdown.so)
    if [[ -d apps/api/sharedLibs/go-html-to-md ]]; then
        info "构建 Go 共享库..."
        cd "$INSTALL_DIR/apps/api/sharedLibs/go-html-to-md"
        go mod download
        go build -o libhtml-to-markdown.so -buildmode=c-shared html-to-markdown.go
        ok "Go 共享库构建完成"
    fi

    # 3. TypeScript API
    if [[ -f apps/api/package.json ]]; then
        info "构建 TypeScript API..."
        cd "$INSTALL_DIR/apps/api"
        npx tsc
        ok "API 构建完成"
    fi

    # 4. Playwright 服务
    if [[ -f apps/playwright-service-ts/package.json ]]; then
        info "构建 Playwright 服务..."
        cd "$INSTALL_DIR/apps/playwright-service-ts"
        npm install
        npx playwright install chromium --with-deps 2>&1 | tail -5
        npx tsc
        ok "Playwright 服务构建完成"
    fi

    info "所有构建完成"
}

# ========== 生成 .env ==========
create_env() {
    local env_file="$INSTALL_DIR/.env"
    info "生成 $env_file ..."

    cat > "$env_file" <<EOF
# ========== Firecrawl Hybrid 部署（中间件 Docker，应用裸机）==========
HOST=0.0.0.0
PORT=$API_PORT
IS_PRODUCTION=true
USE_DB_AUTHENTICATION=false

# Redis
REDIS_URL=redis://localhost:6379

# NUQ 队列（PostgreSQL + RabbitMQ）
NUQ_DATABASE_URL=postgresql://${PG_USER}:${PG_PASS}@localhost:5432/${PG_DB}
NUQ_DATABASE_URL_LISTEN=postgresql://${PG_USER}:${PG_PASS}@localhost:5432/${PG_DB}?application_name=firecrawl-nuq
NUQ_RABBITMQ_URL=amqp://guest:guest@localhost:5672

# Playwright 渲染服务
PLAYWRIGHT_MICROSERVICE_URL=http://localhost:${PLAYWRIGHT_PORT}

# Worker 配置
NUQ_WORKER_COUNT=${NUQ_WORKER_COUNT}

# 认证密钥（按需修改）
BULL_AUTH_KEY=$(openssl rand -hex 16)

# 性能限制（0~1）
MAX_RAM=0.95
MAX_CPU=0.95
EOF

    chmod 0640 "$env_file"
    ok ".env 已生成（Bull 看板密钥: $(grep BULL_AUTH_KEY "$env_file" | cut -d= -f2)）"
}

# ========== 创建 systemd 服务 ==========
create_systemd_services() {
    info "创建 systemd 服务..."

    # ---------- Playwright 服务 ----------
    cat > /etc/systemd/system/firecrawl-playwright.service <<'EOF'
[Unit]
Description=Firecrawl Playwright Rendering Service
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/firecrawl/apps/playwright-service-ts
ExecStart=/usr/bin/node dist/api.js
Restart=always
RestartSec=5
Environment=PORT=3003
Environment=BLOCK_MEDIA=true
Environment=NODE_ENV=production

[Install]
WantedBy=multi-user.target
EOF

    # ---------- Firecrawl 主服务（API + 所有 Workers） ----------
    cat > /etc/systemd/system/firecrawl.service <<'SERVICEEOF'
[Unit]
Description=Firecrawl API + Workers
After=network.target docker.service firecrawl-playwright.service
Wants=docker.service firecrawl-playwright.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/firecrawl/apps/api
ExecStart=/usr/bin/pnpm harness --start-built
Restart=on-failure
RestartSec=10
TimeoutStopSec=60

EnvironmentFile=/opt/firecrawl/.env
Environment=PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
SERVICEEOF

    systemctl daemon-reload
    ok "systemd 服务创建完成"
}

# ========== 启动 ==========
start_services() {
    info "启动服务..."

    systemctl enable firecrawl-playwright.service --now 2>&1 | head -5
    sleep 2

    systemctl enable firecrawl.service --now 2>&1 | head -5

    info "等待服务就绪..."
    for i in $(seq 1 30); do
        if curl -sf "http://localhost:${API_PORT}/health" &>/dev/null; then
            ok "API 服务就绪 http://localhost:${API_PORT}"
            break
        fi
        sleep 2
    done

    if curl -sf "http://localhost:${PLAYWRIGHT_PORT}/health" &>/dev/null; then
        ok "Playwright 渲染服务就绪 http://localhost:${PLAYWRIGHT_PORT}"
    else
        warn "Playwright 服务可能还在启动中，稍后查看 systemctl status firecrawl-playwright"
    fi

    echo ""
    echo -e "${CYAN}==================================${NC}"
    echo -e "${CYAN}  部署完成${NC}"
    echo -e "${CYAN}==================================${NC}"
    echo ""
    echo "  API 地址:      http://<服务器IP>:${API_PORT}"
    echo "  Playwright:    http://localhost:${PLAYWRIGHT_PORT}"
    echo "  Bull 看板:     http://<服务器IP>:${API_PORT}/admin/queues"
    echo "  Bull 密钥:     $(grep BULL_AUTH_KEY /opt/firecrawl/.env | cut -d= -f2)"
    echo "  RabbitMQ 管理: http://localhost:15672 (guest/guest)"
    echo ""
    echo "  常用命令:"
    echo "    systemctl status firecrawl              - API + Workers"
    echo "    systemctl status firecrawl-playwright   - 渲染服务"
    echo "    journalctl -u firecrawl -f              - 实时日志"
    echo ""
    echo "  中间件容器管理:"
    echo "    cd /opt/firecrawl/deploy && docker compose -f docker-compose-infra.yml ps"
    echo "    cd /opt/firecrawl/deploy && docker compose -f docker-compose-infra.yml logs -f"
    echo ""
    echo "  ./deploy.sh 可重复运行（幂等），自动拉取最新代码重建"
    echo "=================================="
}

# ========== 主流程 ==========
main() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}  Firecrawl 一键部署（混合架构）${NC}"
    echo -e "${CYAN}  中间件 → Docker  |  应用 → 裸机  ${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""

    precheck
    install_system_deps
    install_nodejs
    install_go
    install_pnpm
    install_docker
    create_compose_file
    start_middleware
    clone_or_pull
    build_all
    create_env
    create_systemd_services
    start_services
}

main "$@"
