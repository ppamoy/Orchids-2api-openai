# Orchids API 多模态功能分析

## 当前情况

### 项目架构
您的项目是一个 **API 代理服务器**，将请求转发到 Orchids 上游服务器：
- 上游地址：`https://orchids-server.calmstone-6964e08a.westeurope.azurecontainerapps.io/agent/coding-agent`
- 上游类型：**Coding Agent**（编程助手）

### 已实现的功能
1. ✅ 文本聊天（通过 `/v1/chat/completions`）
2. ✅ 多账号管理和负载均衡
3. ✅ 路由已修复（图像和视频端点可访问）

### 问题分析

#### 核心问题
当前的 `GenerateImage` 和 `GenerateVideo` 方法都向 **同一个 coding-agent 端点** 发送请求，但：

1. **端点不匹配**：
   - 发送的是：`{"prompt": "...", "size": "...", "n": 1}`
   - 期望的是：`AgentRequest` 格式（包含 projectId, userId, email 等）

2. **功能不确定**：
   - Orchids Agent 声称支持图像、视频、音乐生成
   - 但不清楚是通过什么方式实现的

## 可能的实现方式

### 方式 1：工具调用（Tool Use）

Orchids Agent 可能通过 **工具调用** 来生成图像和视频：

```
用户请求 → Agent 理解意图 → 调用图像生成工具 → 返回图像 URL
```

**特征**：
- 响应中会包含 `tool_use` 或 `tool_calls` 字段
- 需要通过自然语言请求："请生成一张图片..."
- Agent 会自动调用相应的工具

### 方式 2：特殊 API 端点

Orchids 可能有专门的图像/视频生成端点：

```
https://orchids-server.../image-generation
https://orchids-server.../video-generation
```

**特征**：
- 需要不同的 URL
- 需要特定的请求格式
- 直接返回图像/视频 URL

### 方式 3：Prompt 触发

通过特定的 prompt 格式触发生成：

```
[IMAGE] A cute cat playing in a garden
[VIDEO] A cat running in slow motion
```

**特征**：
- 使用特殊标记或格式
- Agent 识别后生成相应内容
- 在响应中返回 URL

## 测试方案

### 测试 1：询问 Agent 能力

```bash
curl -X POST http://192.168.1.201:3002/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4-5",
    "messages": [{
      "role": "user",
      "content": "What are your capabilities? Can you generate images or videos?"
    }],
    "stream": false
  }'
```

**预期结果**：
- 如果支持：Agent 会说明它可以生成图像/视频
- 如果不支持：Agent 会说它只能编写代码

### 测试 2：请求生成图像

```bash
curl -X POST http://192.168.1.201:3002/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4-5",
    "messages": [{
      "role": "user",
      "content": "Please generate an image of a cute cat. I need the image URL."
    }],
    "stream": false
  }'
```

**预期结果**：
- 如果支持工具调用：响应中会有 `tool_use` 字段和图像 URL
- 如果不支持：Agent 会说它不能生成图像

### 测试 3：检查工具列表

```bash
curl -X POST http://192.168.1.201:3002/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-opus-4-5",
    "messages": [{
      "role": "user",
      "content": "List all the tools you have access to."
    }],
    "stream": false
  }'
```

**预期结果**：
- 如果有工具：Agent 会列出可用的工具
- 可能包括：`generate_image`, `generate_video`, `generate_music` 等

## 解决方案

### 方案 A：如果 Orchids 支持工具调用

**实现步骤**：

1. **修改图像生成处理器**：
```go
func (h *Handler) HandleOpenAIImages(w http.ResponseWriter, r *http.Request) {
    // 解析请求
    var req OpenAIImageRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // 构建自然语言请求
    prompt := fmt.Sprintf("Please generate an image with the following description: %s. Size: %s. Return only the image URL.", req.Prompt, req.Size)
    
    // 发送到 Agent
    claudeReq := OpenAIRequest{
        Model: "claude-opus-4-5",
        Messages: []OpenAIMessage{{
            Role: "user",
            Content: prompt,
        }},
        Stream: false,
    }
    
    // 调用 Agent 并解析响应
    // 从响应中提取图像 URL
    // 返回 OpenAI 格式的响应
}
```

2. **解析工具调用响应**：
   - 检查响应中的 `tool_use` 字段
   - 提取图像 URL
   - 转换为 OpenAI 格式

### 方案 B：如果 Orchids 有专门端点

**实现步骤**：

1. **查找正确的端点**：
   - 联系 Orchids 官方
   - 查看官方文档
   - 或通过抓包分析 Orchids 网页版

2. **更新 Client 代码**：
```go
const (
    chatURL  = "https://orchids-server.../agent/coding-agent"
    imageURL = "https://orchids-server.../image-generation"
    videoURL = "https://orchids-server.../video-generation"
)

func (c *Client) GenerateImage(ctx context.Context, prompt string, size string) (string, error) {
    // 使用 imageURL 而不是 chatURL
    req, _ := http.NewRequestWithContext(ctx, "POST", imageURL, ...)
    // ...
}
```

### 方案 C：如果 Orchids 不支持多模态

**替代方案**：

1. **集成第三方服务**：
   - DALL-E 3 (OpenAI)
   - Stable Diffusion
   - Midjourney API
   - Runway (视频)

2. **修改架构**：
```go
func (h *Handler) HandleOpenAIImages(w http.ResponseWriter, r *http.Request) {
    // 不调用 Orchids
    // 直接调用第三方图像生成 API
    imageURL, err := generateImageWithDALLE(req.Prompt, req.Size)
    // 返回结果
}
```

## 下一步行动

### 立即执行

1. **运行测试脚本**：
```powershell
# 手动测试（避免编码问题）
$payload = '{"model":"claude-opus-4-5","messages":[{"role":"user","content":"Can you generate images?"}],"stream":false}'
Invoke-RestMethod -Uri "http://192.168.1.201:3002/v1/chat/completions" -Method Post -Body $payload -ContentType "application/json" -TimeoutSec 120
```

2. **分析响应**：
   - 查看 Agent 的回答
   - 检查是否有 `tool_use` 字段
   - 确认是否支持多模态

3. **根据结果选择方案**：
   - 如果支持 → 实现方案 A
   - 如果有专门端点 → 实现方案 B
   - 如果不支持 → 实现方案 C

### 需要的信息

1. **Orchids 官方文档**：
   - API 文档链接
   - 工具列表
   - 示例代码

2. **账号测试**：
   - 在 Orchids 网页版测试图像生成
   - 观察网络请求
   - 记录请求格式和端点

3. **技术支持**：
   - 联系 Orchids 官方
   - 询问 API 能力
   - 获取集成指南

## 总结

**当前状态**：
- ✅ 路由已修复
- ✅ 端点可访问
- ⚠️ 功能实现待确认

**关键问题**：
- Orchids Agent 是否真的支持图像/视频生成？
- 如果支持，通过什么方式？
- 需要什么样的请求格式？

**建议**：
1. 先测试 Agent 的实际能力
2. 根据测试结果选择实现方案
3. 如果 Orchids 不支持，考虑集成第三方服务

---

**文档创建时间**：2026-02-01
**状态**：待测试验证
