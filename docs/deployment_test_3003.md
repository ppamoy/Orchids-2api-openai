# Orchids API 3003端口部署测试报告

## 部署信息
- **部署时间**: 2026-02-01 16:53
- **部署方式**: 从GitHub重新克隆干净代码
- **仓库地址**: https://github.com/eslxxx/Orchids-2api-openai.git
- **部署端口**: 3003
- **容器名称**: orchids-3003-orchids-api-1
- **部署目录**: /mnt/docker/orchids-3003

## 部署状态
✅ **部署成功**
- 容器状态: Running (healthy)
- 健康检查: http://192.168.1.201:3003/health - 返回 200 OK
- 服务启动正常

## API测试结果

### 1. 健康检查端点
```bash
GET http://192.168.1.201:3003/health
```
**结果**: ✅ 成功
```json
{"status":"ok"}
```

### 2. 模型列表端点
```bash
GET http://192.168.1.201:3003/v1/models
```
**结果**: ✅ 成功 - 返回6个可用模型

### 3. 聊天补全端点 (非流式)
```bash
POST http://192.168.1.201:3003/v1/chat/completions
{
  "model": "claude-opus-4-5",
  "messages": [{"role": "user", "content": "你好"}],
  "stream": false
}
```
**结果**: ❌ 失败
- 错误: 请求超时 (90秒)
- 日志显示: "Error: context canceled"

### 4. 聊天补全端点 (流式)
```bash
POST http://192.168.1.201:3003/v1/chat/completions
{
  "model": "claude-opus-4-5",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}
```
**结果**: ⚠️ 部分成功
- HTTP状态码: 200
- 返回SSE流,但**内容为空**
- 响应示例:
```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":xxx,"model":"claude-opus-4-5","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":xxx,"model":"claude-opus-4-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}
data: [DONE]
```

## 问题分析

### 核心问题: 上游API返回空内容

**日志证据**:
```
2026/02/01 09:29:50 使用账号: 12223 (duongthienlambbbe5eqw7te7@nbspace.us)
2026/02/01 09:29:50 模型映射: claude-opus-4-5 -> claude-opus-4.5
2026/02/01 09:29:50 新请求进入 (OpenAI格式)
2026/02/01 09:29:50 Error: stream error: stream ID 5; NO_ERROR; received from peer
2026/02/01 09:29:50 请求完成: 输入=99 tokens, 输出=0 tokens, 耗时=1m54s
```

**关键发现**:
1. ✅ 账号认证成功 (账号ID: 12223)
2. ✅ 模型映射正常
3. ✅ 请求发送到上游成功
4. ❌ **输出=0 tokens** - 上游没有返回任何文本内容
5. ❌ **stream error: NO_ERROR** - 上游提前关闭连接

### 可能原因

1. **账号配额问题**
   - 账号可能已用完配额
   - 账号可能被限流或暂停

2. **上游服务问题**
   - Orchids上游API可能有故障
   - 特定模型可能不可用

3. **账号权限问题**
   - 账号可能没有访问某些模型的权限
   - 账号可能需要重新激活

## 测试的模型

所有模型都返回空内容:
- ❌ claude-opus-4-5
- ❌ claude-sonnet-4-5
- ❌ gpt-4
- ❌ gpt-4o

## 建议

1. **检查账号状态**
   - 登录Orchids管理后台查看账号配额
   - 确认账号是否正常激活
   - 检查是否有使用限制

2. **测试其他账号**
   - 如果有其他账号,可以添加并测试
   - 对比不同账号的表现

3. **联系Orchids支持**
   - 如果账号状态正常但仍无法使用
   - 可能需要联系Orchids技术支持

4. **监控上游服务**
   - 检查Orchids官方是否有服务公告
   - 确认上游API是否正常运行

## 服务管理命令

### 查看日志
```powershell
wsl sshpass -p lyskp123 ssh root@192.168.1.201 "cd /mnt/docker/orchids-3003 ; docker-compose logs -f"
```

### 重启服务
```powershell
wsl sshpass -p lyskp123 ssh root@192.168.1.201 "cd /mnt/docker/orchids-3003 ; docker-compose restart"
```

### 停止服务
```powershell
wsl sshpass -p lyskp123 ssh root@192.168.1.201 "cd /mnt/docker/orchids-3003 ; docker-compose down"
```

### 查看容器状态
```powershell
wsl sshpass -p lyskp123 ssh root@192.168.1.201 "cd /mnt/docker/orchids-3003 ; docker-compose ps"
```

## 结论

✅ **部署成功** - 服务正常运行,容器健康
❌ **功能异常** - 上游API返回空内容,无法正常使用

**根本原因**: 上游Orchids API返回空响应,可能是账号配额或权限问题

**下一步**: 需要检查Orchids账号状态和配额
