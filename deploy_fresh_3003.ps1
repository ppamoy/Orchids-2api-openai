# 从GitHub重新部署Orchids API到NAS的3003端口

# NAS配置
$NAS_IP = "192.168.1.201"
$NAS_USER = "root"
$NAS_PASS = "lyskp123"
$NAS_DIR = "/mnt/docker/orchids-3003"
$GITHUB_REPO = "https://github.com/eslxxx/Orchids-2api-openai.git"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "从GitHub重新部署Orchids API到3003端口" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 检查WSL是否可用
Write-Host "`n[1/6] 检查WSL环境..." -ForegroundColor Yellow
try {
    $wslTest = wsl --list --quiet
    Write-Host "OK WSL环境正常" -ForegroundColor Green
}
catch {
    Write-Host "ERROR WSL未安装或不可用" -ForegroundColor Red
    exit 1
}

# 测试NAS连接
Write-Host "`n[2/6] 测试NAS连接..." -ForegroundColor Yellow
$testCmd = "wsl sshpass -p $NAS_PASS ssh -o StrictHostKeyChecking=no $NAS_USER@$NAS_IP `"echo OK`""
$testResult = Invoke-Expression $testCmd
if ($testResult -match "OK") {
    Write-Host "OK NAS连接成功" -ForegroundColor Green
}
else {
    Write-Host "ERROR 无法连接到NAS" -ForegroundColor Red
    exit 1
}

# 停止并删除旧容器
Write-Host "`n[3/6] 清理旧部署..." -ForegroundColor Yellow
$cleanCmd = "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"cd $NAS_DIR ; docker-compose down ; cd .. ; rm -rf orchids-3003 ; echo Done`""
Invoke-Expression $cleanCmd | Out-Null
Write-Host "OK 清理完成" -ForegroundColor Green

# 在NAS上克隆GitHub仓库
Write-Host "`n[4/6] 从GitHub克隆代码到NAS..." -ForegroundColor Yellow
$cloneCmd = "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"mkdir -p $NAS_DIR ; cd $NAS_DIR ; git clone $GITHUB_REPO . ; echo Done`""
$cloneResult = Invoke-Expression $cloneCmd
Write-Host "OK 代码克隆完成" -ForegroundColor Green

# 创建docker-compose.yml
Write-Host "`n[5/6] 配置Docker环境..." -ForegroundColor Yellow

# 创建临时文件
$tempCompose = [System.IO.Path]::GetTempFileName()
$tempDockerfile = [System.IO.Path]::GetTempFileName()

# 写入docker-compose.yml内容
@'
version: '3'
services:
  orchids-api:
    build: .
    ports:
      - "3003:3003"
    environment:
      - PORT=3003
      - DEBUG_ENABLED=false
      - OPENAI_KEY=${OPENAI_KEY}
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3003/health"]
      interval: 30s
      timeout: 10s
      retries: 3
'@ | Out-File -FilePath $tempCompose -Encoding UTF8

# 写入Dockerfile内容
@'
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

EXPOSE 3003

ENV PORT=3003
ENV DEBUG_ENABLED=false

CMD ["./orchids-api"]
'@ | Out-File -FilePath $tempDockerfile -Encoding UTF8

# 转换为WSL路径
$wslCompose = wsl wslpath -a $tempCompose
$wslDockerfile = wsl wslpath -a $tempDockerfile

# 上传文件到NAS
wsl sshpass -p $NAS_PASS scp $wslCompose "$NAS_USER@$NAS_IP`:$NAS_DIR/docker-compose.yml"
wsl sshpass -p $NAS_PASS scp $wslDockerfile "$NAS_USER@$NAS_IP`:$NAS_DIR/Dockerfile"

# 删除临时文件
Remove-Item $tempCompose
Remove-Item $tempDockerfile

Write-Host "OK Docker配置文件已创建" -ForegroundColor Green

# 构建并启动容器
Write-Host "`n[6/6] 构建并启动Docker容器..." -ForegroundColor Yellow
Write-Host "这可能需要几分钟时间..." -ForegroundColor Gray
$buildCmd = "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"cd $NAS_DIR ; docker-compose up -d --build`""
Invoke-Expression $buildCmd

# 等待容器启动
Write-Host "`n等待容器启动..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# 检查部署状态
Write-Host "`n检查部署状态..." -ForegroundColor Yellow
$statusCmd = "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"cd $NAS_DIR ; docker-compose ps`""
Invoke-Expression $statusCmd

# 检查容器日志
Write-Host "`n最近的容器日志:" -ForegroundColor Yellow
$logsCmd = "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"cd $NAS_DIR ; docker-compose logs --tail=20`""
Invoke-Expression $logsCmd

# 显示部署结果
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "部署完成！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "API地址: http://$($NAS_IP):3003" -ForegroundColor White
Write-Host "健康检查: http://$($NAS_IP):3003/health" -ForegroundColor White
Write-Host "`n查看实时日志:" -ForegroundColor Yellow
Write-Host "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"cd $NAS_DIR ; docker-compose logs -f`"" -ForegroundColor Gray
Write-Host "`n停止服务:" -ForegroundColor Yellow
Write-Host "wsl sshpass -p $NAS_PASS ssh $NAS_USER@$NAS_IP `"cd $NAS_DIR ; docker-compose down`"" -ForegroundColor Gray
