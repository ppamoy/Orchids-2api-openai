package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"orchids-api/internal/client"
	"orchids-api/internal/debug"
	"orchids-api/internal/prompt"
	"orchids-api/internal/tiktoken"
)

// OpenAI 请求格式
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type OpenAIMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // 可以是 string 或 []OpenAIContentPart
}

// OpenAI 多模态内容
type OpenAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// OpenAI 响应格式
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage,omitempty"`
}

type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      *OpenAIMessage `json:"message,omitempty"`
	Delta        *OpenAIDelta   `json:"delta,omitempty"`
	FinishReason *string        `json:"finish_reason"`
}

type OpenAIDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAI 流式响应块
type OpenAIStreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
}

// HandleOpenAIChat 处理 OpenAI 格式的 /v1/chat/completions 请求
func (h *Handler) HandleOpenAIChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 转换 OpenAI messages 到 Claude messages
	claudeMessages := make([]prompt.Message, 0, len(req.Messages))
	var systemContent string

	for _, msg := range req.Messages {
		// 解析 content（可能是 string 或 array）
		var textContent string
		var contentBlocks []prompt.ContentBlock

		switch content := msg.Content.(type) {
		case string:
			textContent = content
		case []interface{}:
			// 多模态内容
			for _, part := range content {
				partMap, ok := part.(map[string]interface{})
				if !ok {
					continue
				}
				partType, _ := partMap["type"].(string)
				switch partType {
				case "text":
					text, _ := partMap["text"].(string)
					contentBlocks = append(contentBlocks, prompt.ContentBlock{
						Type: "text",
						Text: text,
					})
				case "image_url":
					if imageURL, ok := partMap["image_url"].(map[string]interface{}); ok {
						url, _ := imageURL["url"].(string)
						// 解析 data URL: data:image/jpeg;base64,xxx
						if strings.HasPrefix(url, "data:") {
							parts := strings.SplitN(url, ",", 2)
							if len(parts) == 2 {
								// 解析 media type
								mediaInfo := strings.TrimPrefix(parts[0], "data:")
								mediaInfo = strings.TrimSuffix(mediaInfo, ";base64")
								base64Data := parts[1]

								contentBlocks = append(contentBlocks, prompt.ContentBlock{
									Type: "image",
									Source: &prompt.ImageSource{
										Type:      "base64",
										MediaType: mediaInfo,
										Data:      base64Data,
									},
								})
							}
						} else {
							// 普通 URL - 需要下载转 base64（暂不支持）
							log.Printf("警告: 不支持 URL 图片，请使用 base64 格式")
						}
					}
				}
			}
		}

		if msg.Role == "system" {
			if textContent != "" {
				systemContent = textContent
			} else {
				// 从 contentBlocks 提取文本
				for _, block := range contentBlocks {
					if block.Type == "text" {
						systemContent += block.Text
					}
				}
			}
			continue
		}

		claudeMsg := prompt.Message{
			Role: msg.Role,
		}

		if len(contentBlocks) > 0 {
			claudeMsg.Content.Blocks = contentBlocks
		} else {
			claudeMsg.Content.Text = textContent
		}

		claudeMessages = append(claudeMessages, claudeMsg)
	}

	// 初始化调试日志
	logger := debug.New(h.config.DebugEnabled)
	defer logger.Close()

	// 选择账号
	var apiClient *client.Client
	var failedAccountIDs []int64

	selectAccount := func() error {
		if h.loadBalancer != nil {
			account, err := h.loadBalancer.GetNextAccountExcluding(failedAccountIDs)
			if err != nil {
				if h.client != nil {
					apiClient = h.client
					return nil
				}
				return err
			}
			log.Printf("使用账号: %s (%s)", account.Name, account.Email)
			apiClient = client.NewFromAccount(account)
			return nil
		} else if h.client != nil {
			apiClient = h.client
			return nil
		}
		return fmt.Errorf("no client configured")
	}

	if err := selectAccount(); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// 构建 prompt
	var systemItems []prompt.SystemItem
	if systemContent != "" {
		systemItems = []prompt.SystemItem{{Type: "text", Text: systemContent}}
	}

	builtPrompt := prompt.BuildPromptV2(prompt.ClaudeAPIRequest{
		Model:    req.Model,
		Messages: claudeMessages,
		System:   systemItems,
		Tools:    nil,
		Stream:   req.Stream,
	})

	// 映射模型
	mappedModel := mapModel(req.Model)
	log.Printf("模型映射: %s -> %s", req.Model, mappedModel)

	// Token 计数
	inputTokens := tiktoken.EstimateTextTokens(builtPrompt)
	var outputTokens int
	var outputMu sync.Mutex

	addOutputTokens := func(text string) {
		if text == "" {
			return
		}
		tokens := tiktoken.EstimateTextTokens(text)
		outputMu.Lock()
		outputTokens += tokens
		outputMu.Unlock()
	}

	msgID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())

	if req.Stream {
		// 流式响应
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		var mu sync.Mutex
		var hasReturn bool
		var fullContent strings.Builder

		writeSSE := func(data string) {
			mu.Lock()
			defer mu.Unlock()
			if hasReturn {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}

		// 发送初始 role delta
		initChunk := OpenAIStreamChunk{
			ID:      msgID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []OpenAIChoice{{
				Index: 0,
				Delta: &OpenAIDelta{Role: "assistant"},
			}},
		}
		initData, _ := json.Marshal(initChunk)
		writeSSE(string(initData))

		log.Println("新请求进入 (OpenAI格式)")

		done := make(chan struct{})

		go func() {
			defer close(done)

			err := apiClient.SendRequest(r.Context(), builtPrompt, []interface{}{}, mappedModel, func(msg client.SSEMessage) {
				mu.Lock()
				if hasReturn {
					mu.Unlock()
					return
				}
				mu.Unlock()

				eventKey := msg.Type
				if msg.Type == "model" && msg.Event != nil {
					if evtType, ok := msg.Event["type"].(string); ok {
						eventKey = "model." + evtType
					}
				}

				switch eventKey {
				case "model.text-delta":
					delta, _ := msg.Event["delta"].(string)
					if delta != "" {
						addOutputTokens(delta)
						fullContent.WriteString(delta)

						chunk := OpenAIStreamChunk{
							ID:      msgID,
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   req.Model,
							Choices: []OpenAIChoice{{
								Index: 0,
								Delta: &OpenAIDelta{Content: delta},
							}},
						}
						data, _ := json.Marshal(chunk)
						writeSSE(string(data))
					}

				case "model.finish":
					finishReason := "stop"
					chunk := OpenAIStreamChunk{
						ID:      msgID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   req.Model,
						Choices: []OpenAIChoice{{
							Index:        0,
							Delta:        &OpenAIDelta{},
							FinishReason: &finishReason,
						}},
					}
					data, _ := json.Marshal(chunk)

					mu.Lock()
					if !hasReturn {
						hasReturn = true
						mu.Unlock()
						fmt.Fprintf(w, "data: %s\n\n", string(data))
						fmt.Fprintf(w, "data: [DONE]\n\n")
						flusher.Flush()
						log.Printf("请求完成: 输入=%d tokens, 输出=%d tokens, 耗时=%v", inputTokens, outputTokens, time.Since(startTime))
					} else {
						mu.Unlock()
					}
				}
			}, logger)

			if err != nil {
				log.Printf("Error: %v", err)
			}

			// 确保发送结束信号
			mu.Lock()
			if !hasReturn {
				hasReturn = true
				mu.Unlock()

				finishReason := "stop"
				chunk := OpenAIStreamChunk{
					ID:      msgID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   req.Model,
					Choices: []OpenAIChoice{{
						Index:        0,
						Delta:        &OpenAIDelta{},
						FinishReason: &finishReason,
					}},
				}
				data, _ := json.Marshal(chunk)

				fmt.Fprintf(w, "data: %s\n\n", string(data))
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()

				log.Printf("请求完成: 输入=%d tokens, 输出=%d tokens, 耗时=%v", inputTokens, outputTokens, time.Since(startTime))
			} else {
				mu.Unlock()
			}
		}()

		<-done

	} else {
		// 非流式响应
		var fullContent strings.Builder

		log.Println("新请求进入 (OpenAI格式，非流式)")

		err := apiClient.SendRequest(r.Context(), builtPrompt, []interface{}{}, mappedModel, func(msg client.SSEMessage) {
			eventKey := msg.Type
			if msg.Type == "model" && msg.Event != nil {
				if evtType, ok := msg.Event["type"].(string); ok {
					eventKey = "model." + evtType
				}
			}

			if eventKey == "model.text-delta" {
				delta, _ := msg.Event["delta"].(string)
				if delta != "" {
					addOutputTokens(delta)
					fullContent.WriteString(delta)
				}
			}
		}, logger)

		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		finishReason := "stop"
		response := OpenAIResponse{
			ID:      msgID,
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []OpenAIChoice{{
				Index: 0,
				Message: &OpenAIMessage{
					Role:    "assistant",
					Content: fullContent.String(),
				},
				FinishReason: &finishReason,
			}},
			Usage: OpenAIUsage{
				PromptTokens:     inputTokens,
				CompletionTokens: outputTokens,
				TotalTokens:      inputTokens + outputTokens,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		log.Printf("请求完成: 输入=%d tokens, 输出=%d tokens, 耗时=%v", inputTokens, outputTokens, time.Since(startTime))
	}
}

// HandleOpenAIModels 处理 /v1/models 请求
func (h *Handler) HandleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	models := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{"id": "claude-opus-4-5", "object": "model", "owned_by": "anthropic"},
			{"id": "claude-opus-4-5-thinking", "object": "model", "owned_by": "anthropic"},
			{"id": "claude-sonnet-4-20250514", "object": "model", "owned_by": "anthropic"},
			{"id": "claude-sonnet-4-5", "object": "model", "owned_by": "anthropic"},
			{"id": "gpt-4", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// OpenAIImageRequest OpenAI 图像生成请求格式
type OpenAIImageRequest struct {
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`
	Size   string `json:"size,omitempty"`
}

// OpenAIImageResponse OpenAI 图像生成响应格式
type OpenAIImageResponse struct {
	Created int64             `json:"created"`
	Data    []OpenAIImageData `json:"data"`
}

type OpenAIImageData struct {
	URL string `json:"url"`
}

// HandleOpenAIImages 处理 /v1/images/generations 请求
func (h *Handler) HandleOpenAIImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OpenAIImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// 选择账号
	var apiClient *client.Client
	var failedAccountIDs []int64

	selectAccount := func() error {
		if h.loadBalancer != nil {
			account, err := h.loadBalancer.GetNextAccountExcluding(failedAccountIDs)
			if err != nil {
				if h.client != nil {
					apiClient = h.client
					return nil
				}
				return err
			}
			log.Printf("使用账号: %s (%s)", account.Name, account.Email)
			apiClient = client.NewFromAccount(account)
			return nil
		} else if h.client != nil {
			apiClient = h.client
			return nil
		}
		return fmt.Errorf("no client configured")
	}

	if err := selectAccount(); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// 设置默认值
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	// 生成图像
	imageURL, err := apiClient.GenerateImage(r.Context(), req.Prompt, size)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		// 尝试使用其他账号
		if h.loadBalancer != nil {
			if err := selectAccount(); err == nil {
				imageURL, err = apiClient.GenerateImage(r.Context(), req.Prompt, size)
			}
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 构建响应
	response := OpenAIImageResponse{
		Created: time.Now().Unix(),
		Data:    []OpenAIImageData{{URL: imageURL}},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Image generated: %s", imageURL)
}

// OpenAIVideoRequest OpenAI 视频生成请求格式
type OpenAIVideoRequest struct {
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`
	Size   string `json:"size,omitempty"`
}

// OpenAIVideoResponse OpenAI 视频生成响应格式
type OpenAIVideoResponse struct {
	Created int64             `json:"created"`
	Data    []OpenAIVideoData `json:"data"`
}

type OpenAIVideoData struct {
	URL string `json:"url"`
}

// HandleOpenAIVideos 处理 /v1/videos/generations 请求
func (h *Handler) HandleOpenAIVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OpenAIVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// 选择账号
	var apiClient *client.Client
	var failedAccountIDs []int64

	selectAccount := func() error {
		if h.loadBalancer != nil {
			account, err := h.loadBalancer.GetNextAccountExcluding(failedAccountIDs)
			if err != nil {
				if h.client != nil {
					apiClient = h.client
					return nil
				}
				return err
			}
			log.Printf("使用账号: %s (%s)", account.Name, account.Email)
			apiClient = client.NewFromAccount(account)
			return nil
		} else if h.client != nil {
			apiClient = h.client
			return nil
		}
		return fmt.Errorf("no client configured")
	}

	if err := selectAccount(); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// 设置默认值
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	// 生成视频
	videoURL, err := apiClient.GenerateVideo(r.Context(), req.Prompt, size)
	if err != nil {
		log.Printf("Error generating video: %v", err)
		// 尝试使用其他账号
		if h.loadBalancer != nil {
			if err := selectAccount(); err == nil {
				videoURL, err = apiClient.GenerateVideo(r.Context(), req.Prompt, size)
			}
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 构建响应
	response := OpenAIVideoResponse{
		Created: time.Now().Unix(),
		Data:    []OpenAIVideoData{{URL: videoURL}},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Video generated: %s", videoURL)
}
