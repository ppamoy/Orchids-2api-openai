#!/bin/bash

# 部署脚本 - 将Orchids API部署到NAS

# NAS配置
NAS_IP="192.168.1.201"
NAS_PORT="3002"
NAS_USER="root"
NAS_PASS="lyskp123"
NAS_DIR="/mnt/docker/orchids"

# 本地配置
LOCAL_DIR="$(pwd)"

# 检查SSH是否可用
echo "检查NAS连接..."
if ! sshpass -p "$NAS_PASS" ssh -o StrictHostKeyChecking=no "$NAS_USER@$NAS_IP" "echo 'Connection test'" > /dev/null 2>&1; then
    echo "错误: 无法连接到NAS，请检查网络连接和凭据"
    exit 1
fi

echo "连接成功！开始部署..."

# 在NAS上创建部署目录
echo "在NAS上创建部署目录..."
sshpass -p "$NAS_PASS" ssh "$NAS_USER@$NAS_IP" "mkdir -p $NAS_DIR"

# 复制项目文件到NAS
echo "复制项目文件到NAS..."
sshpass -p "$NAS_PASS" scp -r "$LOCAL_DIR/*" "$NAS_USER@$NAS_IP:$NAS_DIR/"

# 创建Dockerfile（如果不存在）
echo "创建Dockerfile..."
sshpass -p "$NAS_PASS" ssh "$NAS_USER@$NAS_IP" "cat > $NAS_DIR/Dockerfile << 'EOF'
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o orchids-api ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/orchids-api /app/
COPY --from=builder /app/data /app/data/
COPY --from=builder /app/web /app/web/

EXPOSE 3002

ENV PORT=3002
ENV DEBUG_ENABLED=false

CMD ["./orchids-api"]
EOF"

# 创建docker-compose.yml
echo "创建docker-compose.yml..."
sshpass -p "$NAS_PASS" ssh "$NAS_USER@$NAS_IP" "cat > $NAS_DIR/docker-compose.yml << 'EOF'
version: '3'
services:
  orchids-api:
    build: .
    ports:
      - "3002:3002"
    environment:
      - PORT=3002
      - DEBUG_ENABLED=false
      - OPENAI_KEY=${OPENAI_KEY}
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3002/health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF"

# 构建和启动Docker容器
echo "构建和启动Docker容器..."
sshpass -p "$NAS_PASS" ssh "$NAS_USER@$NAS_IP" "cd $NAS_DIR && docker-compose up -d --build"

# 检查部署状态
echo "检查部署状态..."
sleep 5
sshpass -p "$NAS_PASS" ssh "$NAS_USER@$NAS_IP" "cd $NAS_DIR && docker-compose ps"

# 显示部署结果
echo "\n部署完成！"
echo "API地址: http://$NAS_IP:$NAS_PORT"
echo "健康检查: http://$NAS_IP:$NAS_PORT/health"
echo "\n使用以下命令查看日志:"
echo "sshpass -p '$NAS_PASS' ssh $NAS_USER@$NAS_IP 'cd $NAS_DIR && docker-compose logs -f'"
