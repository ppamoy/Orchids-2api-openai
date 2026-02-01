# Orchids API 项目文档

## 1. 项目概述

Orchids API 是一个通用多模态 AI 接口服务，支持文本、图像和视频生成功能。该项目基于 Go 语言开发，提供了与 OpenAI 兼容的 API 接口，可与多种 AI 模型集成。

### 1.1 核心功能

- **文本聊天**：支持与多种 AI 模型进行对话，包括 Claude 和 GPT 系列模型
- **图像生成**：根据文本描述生成高质量图像
- **视频生成**：根据文本描述生成视频内容
- **模型管理**：提供模型列表和详细信息查询
- **账号负载均衡**：支持多账号轮询和故障转移

### 1.2 技术栈

- **后端**：Go 语言
- **数据库**：SQLite
- **容器化**：Docker
- **部署**：NAS (Network Attached Storage)

## 2. 项目结构

```
orchids/
├── cmd/
│   └── server/           # 服务器入口
├── internal/
│   ├── api/              # API 处理
│   ├── client/           # 客户端实现
│   ├── config/           # 配置管理
│   ├── debug/            # 调试工具
│   ├── handler/          # 请求处理器
│   ├── loadbalancer/     # 负载均衡
│   ├── middleware/       # 中间件
│   └── store/            # 数据存储
├── web/
│   └── static/           # 静态文件
├── data/                 # 数据目录
├── Dockerfile            # Docker 构建文件
├── docker-compose.yml    # Docker Compose 配置
├── deploy_wsl.sh         # 部署脚本
└── PROJECT_DOCUMENTATION.md  # 项目文档
```

## 3. API 接口

### 3.1 文本聊天接口

**端点**：`/v1/chat/completions`

**方法**：POST

**请求体**：

```json
{
  "model": "claude-opus-4-5",
  "messages": [
    {
      "role": "user",
      "content": "你好，能帮我写一首诗吗？"
    }
  ],
  "stream": true
}
```

**响应**：

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1738456789,
  "model": "claude-opus-4-5",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "当然可以，以下是一首关于春天的诗..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 50,
    "total_tokens": 60
  }
}
```

### 3.2 图像生成接口

**端点**：`/v1/images/generations`

**方法**：POST

**请求体**：

```json
{
  "prompt": "一只可爱的小猫在花园里玩耍",
  "size": "1024x1024",
  "n": 1
}
```

**响应**：

```json
{
  "created": 1738456789,
  "data": [
    {
      "url": "https://example.com/image.png"
    }
  ]
}
```

### 3.3 视频生成接口

**端点**：`/v1/videos/generations`

**方法**：POST

**请求体**：

```json
{
  "prompt": "一只可爱的小猫在花园里玩耍",
  "size": "1024x1024",
  "n": 1
}
```

**响应**：

```json
{
  "created": 1738456789,
  "data": [
    {
      "url": "https://example.com/video.mp4"
    }
  ]
}
```

### 3.4 模型列表接口

**端点**：`/v1/models`

**方法**：GET

**响应**：

```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-opus-4-5",
      "object": "model",
      "created": 1738456789,
      "owned_by": "anthropic"
    },
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1738456789,
      "owned_by": "openai"
    }
  ]
}
```

## 4. 配置管理

### 4.1 环境变量

项目通过环境变量进行配置，支持以下配置项：

| 环境变量 | 描述 | 默认值 |
|---------|------|-------|
| `PORT` | 服务器端口 | "3002" |
| `DEBUG_ENABLED` | 是否启用调试模式 | "false" |
| `ADMIN_USER` | 管理员用户名 | "admin" |
| `ADMIN_PASS` | 管理员密码 | "password" |
| `ADMIN_PATH` | 管理员路径 | "/admin" |
| `OPENAI_KEY` | OpenAI API 密钥 | "" |

### 4.2 配置文件

项目支持通过 `.env` 文件加载环境变量，文件格式如下：

```env
# 服务器配置
PORT=3002
DEBUG_ENABLED=false

# 管理员配置
ADMIN_USER=admin
ADMIN_PASS=your_secure_password
ADMIN_PATH=/admin

# API 密钥
OPENAI_KEY=your_openai_api_key
```

## 5. 部署指南

### 5.1 系统要求

- **Docker**：支持 Docker 19.03+ 版本
- **Docker Compose**：支持 Docker Compose 1.25+ 版本
- **NAS**：支持运行 Docker 的网络存储设备
- **网络**：NAS 需在局域网内可访问

### 5.2 NAS 部署步骤

#### 5.2.1 准备工作

1. **确保 NAS 已安装 Docker**：
   - 登录 NAS 管理界面
   - 进入应用中心
   - 安装 Docker 应用

2. **配置 NAS 网络**：
   - 确保 NAS 有固定 IP 地址（推荐）
   - 确保端口 3002 未被占用

#### 5.2.2 自动化部署

项目提供了 `deploy_wsl.sh` 脚本，可通过 WSL 在 Windows 环境下自动化部署到 NAS：

**使用方法**：

1. **配置部署参数**：
   编辑 `deploy_wsl.sh` 文件，修改以下参数：
   ```bash
   # NAS 连接信息
   NAS_IP="192.168.1.201"
   NAS_USER="root"
   NAS_PASS="lyskp123"
   NAS_DIR="/mnt/docker/orchids"
   ```

2. **执行部署脚本**：
   ```bash
   # 在项目根目录执行
   wsl -e bash ./deploy_wsl.sh
   ```

3. **部署过程**：
   - 检查 NAS 连接
   - 创建部署目录
   - 复制项目文件
   - 创建 Docker 配置文件
   - 构建和启动容器

#### 5.2.3 手动部署

如果需要手动部署，可按照以下步骤操作：

1. **复制项目文件到 NAS**：
   ```bash
   scp -r ./cmd ./internal ./web ./data ./Dockerfile ./docker-compose.yml root@192.168.1.201:/mnt/docker/orchids/
   ```

2. **创建 .env 文件**：
   ```bash
   ssh root@192.168.1.201 'cat > /mnt/docker/orchids/.env << EOF
   PORT=3002
   DEBUG_ENABLED=false
   ADMIN_USER=admin
   ADMIN_PASS=your_secure_password
   ADMIN_PATH=/admin
   OPENAI_KEY=your_openai_api_key
   EOF'
   ```

3. **构建和启动容器**：
   ```bash
   ssh root@192.168.1.201 'cd /mnt/docker/orchids && docker-compose up -d --build'
   ```

### 5.3 部署验证

部署完成后，可通过以下方式验证服务状态：

1. **健康检查**：
   ```bash
   curl http://192.168.1.201:3002/health
   ```

2. **测试页面**：
   打开浏览器访问：`http://192.168.1.201:3002/static/test.html`

3. **查看日志**：
   ```bash
   ssh root@192.168.1.201 'cd /mnt/docker/orchids && docker-compose logs -f'
   ```

### 5.4 部署架构

**部署架构图**：

```
┌─────────────────┐
│   客户端浏览器   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   局域网网络     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     NAS 设备    │
├─────────────────┤
│  ┌───────────┐  │
│  │ Docker    │  │
│  │ Container │  │
│  │  ┌──────┐ │  │
│  │  │ Go   │ │  │
│  │  │ API  │ │  │
│  │  └──────┘ │  │
│  └───────────┘  │
│                 │
│  ┌───────────┐  │
│  │ SQLite   │  │
│  │ Database │  │
│  └───────────┘  │
└─────────────────┘
```

## 6. 测试指南

### 6.1 测试页面

项目提供了 `static/test.html` 测试页面，可通过浏览器访问进行功能测试：

**访问地址**：`http://192.168.1.201:3002/static/test.html`

**测试页面功能**：

1. **文字聊天**：测试文本对话功能，支持选择不同模型
2. **图片生成**：测试图像生成功能，支持选择不同尺寸
3. **视频生成**：测试视频生成功能，支持选择不同尺寸

### 6.2 API 测试

可使用 curl 或 Postman 等工具进行 API 测试：

**示例 1：文本聊天**

```bash
curl -X POST http://192.168.1.201:3002/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4-5",
    "messages": [
      {
        "role": "user",
        "content": "你好，请介绍一下你自己"
      }
    ],
    "stream": false
  }'
```

**示例 2：图像生成**

```bash
curl -X POST http://192.168.1.201:3002/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "一只可爱的小猫在花园里玩耍",
    "size": "1024x1024",
    "n": 1
  }'
```

**示例 3：视频生成**

```bash
curl -X POST http://192.168.1.201:3002/v1/videos/generations \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "一只可爱的小猫在花园里玩耍",
    "size": "1024x1024",
    "n": 1
  }'
```

## 7. 故障排查

### 7.1 常见问题

**问题 1：服务启动失败**

- **可能原因**：端口被占用、配置文件错误、Docker 权限问题
- **排查方法**：
  ```bash
  # 查看容器状态
  ssh root@192.168.1.201 'docker ps -a'
  
  # 查看容器日志
  ssh root@192.168.1.201 'docker logs orchids-orchids-api-1'
  ```

**问题 2：API 请求返回错误**

- **可能原因**：API 密钥无效、模型不可用、网络连接问题
- **排查方法**：
  ```bash
  # 查看应用日志
  ssh root@192.168.1.201 'cd /mnt/docker/orchids && docker-compose logs -f'
  ```

**问题 3：测试页面 404**

- **可能原因**：静态文件路径错误、服务未正确部署
- **排查方法**：
  ```bash
  # 检查静态文件是否存在
  ssh root@192.168.1.201 'ls -la /mnt/docker/orchids/web/static/'
  ```

### 7.2 日志管理

项目日志存储在容器内部，可通过以下方式访问：

```bash
# 查看最近日志
ssh root@192.168.1.201 'cd /mnt/docker/orchids && docker-compose logs --tail=100'

# 实时查看日志
ssh root@192.168.1.201 'cd /mnt/docker/orchids && docker-compose logs -f'
```

## 8. 维护与更新

### 8.1 版本更新

**步骤**：

1. **获取最新代码**：
   ```bash
   git pull origin master
   ```

2. **重新部署**：
   ```bash
   wsl -e bash ./deploy_wsl.sh
   ```

### 8.2 数据备份

**建议定期备份以下数据**：

1. **数据库文件**：`data/orchids.db`
2. **配置文件**：`.env`
3. **日志文件**：（如启用了调试模式）

**备份方法**：

```bash
# 从 NAS 复制到本地
scp root@192.168.1.201:/mnt/docker/orchids/data/orchids.db ./backup/
scp root@192.168.1.201:/mnt/docker/orchids/.env ./backup/
```

### 8.3 性能优化

**优化建议**：

1. **增加账号数量**：添加多个 API 账号以提高并发处理能力
2. **调整模型参数**：根据实际需求调整模型参数，平衡质量和速度
3. **网络优化**：确保 NAS 网络连接稳定，考虑使用有线网络连接

## 9. 安全注意事项

### 9.1 安全建议

1. **API 密钥保护**：
   - 不要在代码中硬编码 API 密钥
   - 使用环境变量或配置文件管理密钥
   - 定期轮换 API 密钥

2. **管理员访问控制**：
   - 使用强密码保护管理员界面
   - 限制管理员界面的访问 IP
   - 定期更改管理员密码

3. **网络安全**：
   - 考虑在生产环境中使用 HTTPS
   - 配置防火墙规则，限制 API 访问
   - 监控异常访问模式

### 9.2 风险评估

| 风险项 | 影响程度 | 缓解措施 |
|-------|---------|--------|
| API 密钥泄露 | 高 | 使用环境变量，定期轮换密钥 |
| 管理员密码泄露 | 中 | 使用强密码，限制访问 IP |
| 网络攻击 | 中 | 配置防火墙，使用 HTTPS |
| 资源耗尽 | 中 | 实现请求限流，监控资源使用 |

## 10. 总结

Orchids API 项目是一个功能完整的多模态 AI 接口服务，支持文本、图像和视频生成。通过 Docker 容器化部署到 NAS，实现了低成本、高可靠性的本地 AI 服务解决方案。

### 10.1 项目优势

- **多模态支持**：集成文本、图像、视频生成功能
- **OpenAI 兼容**：提供与 OpenAI 一致的 API 接口
- **本地部署**：基于 NAS 本地部署，数据隐私可控
- **易于扩展**：模块化设计，支持添加新模型和功能
- **负载均衡**：支持多账号轮询，提高可靠性

### 10.2 应用场景

- **个人助手**：提供智能问答和内容生成服务
- **内容创作**：辅助生成文章、图像和视频内容
- **教育工具**：作为学习辅助工具，提供知识查询和解释
- **开发测试**：为开发人员提供 AI 接口测试环境

### 10.3 未来规划

- **支持更多模型**：集成更多 AI 模型，如开源模型
- **增强功能**：添加语音识别和合成功能
- **用户管理**：实现多用户系统，支持 API 密钥管理
- **监控系统**：添加性能监控和告警功能
- **文档优化**：提供更详细的 API 文档和使用示例

---

**文档更新时间**：2026-02-01
**版本**：1.0.0
