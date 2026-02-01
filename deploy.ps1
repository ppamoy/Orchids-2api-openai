# 部署脚本 - 将Orchids API部署到NAS

# NAS配置
$NAS_IP = "192.168.1.201"
$NAS_USER = "root"
$NAS_PASS = "lyskp123"
$NAS_DIR = "/mnt/docker/orchids"

# 本地配置
$LOCAL_DIR = Get-Location

Write-Host "检查NAS连接..."
try {
    # 测试SSH连接
    $test = sshpass -p $NAS_PASS ssh -o StrictHostKeyChecking=no "$NAS_USER@$NAS_IP" "echo 'Connection test'"
    Write-Host "连接成功！开始部署..."
} catch {
    Write-Host "错误: 无法连接到NAS，请检查网络连接和凭据"
    exit 1
}

# 在NAS上创建部署目录
Write-Host "在NAS上创建部署目录..."
sshpass -p $NAS_PASS ssh "$NAS_USER@$NAS_IP" "mkdir -p $NAS_DIR"

# 复制项目文件到NAS
Write-Host "复制项目文件到NAS..."
$files = @(
    "go.mod",
    "go.sum",
    "cmd",
    "internal",
    "web",
    "data"
)

foreach ($file in $files) {
    $localPath = Join-Path $LOCAL_DIR $file
    if (Test-Path $localPath) {
        Write-Host "复制 $file ..."
        sshpass -p $NAS_PASS scp -r "$localPath" "$NAS_USER@$NAS_IP`:$NAS_DIR/"
    } else {
        Write-Host "警告: $file 不存在，跳过"
    }
}

# 创建Dockerfile
Write-Host "创建Dockerfile..."
$dockerfileContent = "FROM golang:1.24-alpine AS builder

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

CMD ["./orchids-api"]"

sshpass -p $NAS_PASS ssh "$NAS_USER@$NAS_IP" "echo '$dockerfileContent' > $NAS_DIR/Dockerfile"

# 创建docker-compose.yml
Write-Host "创建docker-compose.yml..."
$dockerComposeContent = "version: '3'
services:
  orchids-api:
    build: .
    ports:
      - \"3002:3002\"
    environment:
      - PORT=3002
      - DEBUG_ENABLED=false
      - OPENAI_KEY=${OPENAI_KEY}
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    healthcheck:
      test: [\"CMD\", \"wget\", \"-qO-\", \"http://localhost:3002/health\"]
      interval: 30s
      timeout: 10s
      retries: 3"

sshpass -p $NAS_PASS ssh "$NAS_USER@$NAS_IP" "echo '$dockerComposeContent' > $NAS_DIR/docker-compose.yml"

# 构建和启动Docker容器
Write-Host "构建和启动Docker容器..."
sshpass -p $NAS_PASS ssh "$NAS_USER@$NAS_IP" "cd $NAS_DIR && docker-compose up -d --build"

# 检查部署状态
Write-Host "检查部署状态..."
Start-Sleep -Seconds 5
sshpass -p $NAS_PASS ssh "$NAS_USER@$NAS_IP" "cd $NAS_DIR && docker-compose ps"

# 显示部署结果
Write-Host "\n部署完成！"
Write-Host "API地址: http://$NAS_IP:3002"
Write-Host "健康检查: http://$NAS_IP:3002/health"
Write-Host "\n使用以下命令查看日志:"
Write-Host "sshpass -p '$NAS_PASS' ssh $NAS_USER@$NAS_IP 'cd $NAS_DIR && docker-compose logs -f'"
