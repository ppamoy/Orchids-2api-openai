# 🚀 快速开始测试

## ✅ 已完成的修复

我已经修复了多模态 API 的路由注册问题：

### 修复内容
在 `cmd/server/main.go` 中添加了两行代码：
```go
mux.HandleFunc("/v1/images/generations", h.HandleOpenAIImages)
mux.HandleFunc("/v1/videos/generations", h.HandleOpenAIVideos)
```

### 修复效果
- ✅ 图像生成端点现在可以访问
- ✅ 视频生成端点现在可以访问
- ✅ 测试页面的所有功能都可以使用

## 📋 立即执行测试（3 步）

### 步骤 1：重新部署服务

```powershell
# 在项目根目录执行
wsl -e bash ./deploy_wsl.sh
```

等待部署完成（约 2-3 分钟）。

### 步骤 2：快速验证

```powershell
# 进入测试目录
cd test

# 执行测试脚本
.\manual_test.ps1
```

### 步骤 3：查看结果

测试脚本会自动测试 5 个端点：
1. ✅ 健康检查
2. ✅ 模型列表
3. ✅ 文本聊天
4. ✅ 图像生成（**新修复**）
5. ✅ 视频生成（**新修复**）

## 🌐 使用测试页面

打开浏览器访问：
```
http://192.168.1.201:3002/static/test.html
```

测试所有三个功能：
- 文字聊天
- 图片生成（**现在可用**）
- 视频生成（**现在可用**）

## 🔍 详细测试（可选）

### 使用 Go 测试
```bash
# 在项目根目录
go test ./test -v
```

### 使用 curl 测试

#### 测试图像生成
```bash
curl -X POST http://192.168.1.201:3002/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{"prompt":"一只可爱的小猫","size":"1024x1024","n":1}'
```

#### 测试视频生成
```bash
curl -X POST http://192.168.1.201:3002/v1/videos/generations \
  -H "Content-Type: application/json" \
  -d '{"prompt":"一只可爱的小猫在玩耍","size":"1024x1024","n":1}'
```

## 📊 预期结果

### 修复前 ❌
```
GET  /v1/images/generations  → 404 Not Found
GET  /v1/videos/generations  → 404 Not Found
```

### 修复后 ✅
```
POST /v1/images/generations  → 200 OK (返回图像 URL)
POST /v1/videos/generations  → 200 OK (返回视频 URL)
```

## 📁 测试文件位置

所有测试相关文件都在 `test/` 目录：

```
test/
├── api_test.go              # Go 自动化测试
├── manual_test.ps1          # PowerShell 测试脚本
├── quick_verify.sh          # Bash 验证脚本
├── deploy_and_test.ps1      # 一键部署和测试
├── test_report_template.md  # 测试报告模板
├── README.md                # 详细测试指南
├── TESTING_SUMMARY.md       # 测试总结
└── testdata/                # 测试数据
    ├── chat_request.json
    ├── image_request.json
    └── video_request.json
```

## 🎯 测试规范文档

完整的测试规范在：
```
.kiro/specs/multimodal-api-testing/
├── requirements.md  # 需求文档
├── design.md        # 设计文档
└── tasks.md         # 任务清单
```

## ⚠️ 注意事项

1. **确保服务已重新部署**：修复只有在重新部署后才会生效
2. **需要有效账号**：图像和视频生成需要配置有效的 API 账号
3. **响应时间**：图像和视频生成可能需要较长时间（10-30 秒）
4. **网络连接**：确保能够访问 NAS 服务器

## 🐛 故障排查

### 问题：测试脚本报错
**解决**：检查服务器地址是否正确，服务是否已启动

### 问题：返回 404
**解决**：确保已重新部署，路由修复已生效

### 问题：返回 503
**解决**：检查是否配置了有效的 API 账号

### 问题：超时
**解决**：图像/视频生成需要时间，可以增加超时设置

## 📞 获取帮助

- 查看 `test/README.md` 了解详细测试指南
- 查看 `test/TESTING_SUMMARY.md` 了解测试总结
- 查看 `PROJECT_DOCUMENTATION.md` 了解项目文档

## 🎉 开始测试！

现在就执行步骤 1，开始测试吧！

```powershell
# 一键部署和测试
cd test
.\deploy_and_test.ps1
```

祝测试顺利！✨
