package service

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"leans/model"
	"net/http"
	"strings"
	"time"
)

type SettingsProvider func() (model.AISettings, error)

type AIService struct {
	defaults  model.AISettings
	settings  SettingsProvider
	client    *http.Client
	maxTokens int
}

func NewAIService(defaults model.AISettings, provider SettingsProvider, maxTokens int) *AIService {
	return &AIService{
		defaults:  defaults,
		settings:  provider,
		maxTokens: maxTokens,
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 90 * time.Second,
				TLSClientConfig: &tls.Config{},
			},
		},
	}
}

func (s *AIService) Settings() (model.AISettings, error) {
	set := s.defaults
	if s.settings != nil {
		persisted, err := s.settings()
		if err != nil {
			return set, err
		}
		for _, field := range []struct{ dst, src *string }{
			{&set.Provider, &persisted.Provider},
			{&set.APIKey, &persisted.APIKey},
			{&set.BaseURL, &persisted.BaseURL},
			{&set.Model, &persisted.Model},
		} {
			if *field.src != "" {
				*field.dst = *field.src
			}
		}
	}
	return set, nil
}

// Validate ensures the resolved settings can be used.
func (s *AIService) Validate() error {
	set, err := s.Settings()
	if err != nil {
		return err
	}
	if set.APIKey == "" {
		return fmt.Errorf("%w: 未配置 API Key，请先在设置中填写", ErrNotConfigured)
	}
	if set.BaseURL == "" {
		return fmt.Errorf("%w: 未配置 API Base URL", ErrNotConfigured)
	}
	if set.Model == "" {
		return fmt.Errorf("%w: 未配置模型名称", ErrNotConfigured)
	}
	return nil
}

// doChat 发送一次 chat/completions 请求，返回原始响应体。
// 设置合并、请求构造、发送与状态码检查统一在此，ChatCompletion 与 Ping 共用。
func (s *AIService) doChat(messages []model.ChatMessage, maxTokens int) ([]byte, error) {
	set, err := s.Settings()
	if err != nil {
		return nil, fmt.Errorf("读取AI配置失败: %w", err)
	}
	if set.APIKey == "" {
		return nil, fmt.Errorf("%w: 未配置 API Key，请先在设置中填写", ErrNotConfigured)
	}
	if set.BaseURL == "" {
		return nil, fmt.Errorf("%w: 未配置 API Base URL", ErrNotConfigured)
	}
	if set.Model == "" {
		return nil, fmt.Errorf("%w: 未配置模型名称", ErrNotConfigured)
	}

	reqBody := model.ChatRequest{
		Model:     set.Model,
		Messages:  messages,
		MaxTokens: maxTokens,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := set.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+set.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 错误 (status %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (s *AIService) ChatCompletion(messages []model.ChatMessage) (string, *model.Usage, error) {
	body, err := s.doChat(messages, s.maxTokens)
	if err != nil {
		return "", nil, err
	}

	var chatResp model.ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, chatResp.Usage, nil
}

// ChatCompletionStream 以 SSE 流式调用 chat/completions，每收到一段内容增量
// 就调用一次 onDelta。返回完整内容、usage 统计，以及从请求发出到首个内容
// 增量的耗时（首字响应时间，毫秒）。
func (s *AIService) ChatCompletionStream(messages []model.ChatMessage, onDelta func(string)) (string, *model.Usage, int64, error) {
	set, err := s.Settings()
	if err != nil {
		return "", nil, 0, fmt.Errorf("读取AI配置失败: %w", err)
	}
	if err := s.Validate(); err != nil {
		return "", nil, 0, err
	}

	reqBody := model.ChatRequest{
		Model:         set.Model,
		Messages:      messages,
		MaxTokens:     s.maxTokens,
		Stream:        true,
		StreamOptions: &model.StreamOptions{IncludeUsage: true},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	url := set.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+set.APIKey)

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, 0, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, 0, fmt.Errorf("API 错误 (status %d): %s", resp.StatusCode, string(body))
	}

	var content strings.Builder
	var usage *model.Usage
	var firstTokenMS int64

	reader := bufio.NewReader(resp.Body)
	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(line[len("data:"):])
			if payload == "[DONE]" {
				break
			}
			var chunk model.ChatStreamChunk
			if json.Unmarshal([]byte(payload), &chunk) == nil {
				if chunk.Usage != nil {
					usage = chunk.Usage
				}
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					if firstTokenMS == 0 {
						firstTokenMS = time.Since(start).Milliseconds()
					}
					content.WriteString(chunk.Choices[0].Delta.Content)
					if onDelta != nil {
						onDelta(chunk.Choices[0].Delta.Content)
					}
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return content.String(), usage, firstTokenMS, fmt.Errorf("读取流式响应失败: %w", readErr)
		}
	}

	if content.Len() == 0 {
		return "", usage, firstTokenMS, fmt.Errorf("流式响应中没有内容")
	}
	return content.String(), usage, firstTokenMS, nil
}

// Ping 用当前配置发一个最小请求，验证 API Key / Base URL / 模型连通性。
// 返回 nil 表示连接成功。
func (s *AIService) Ping() error {
	_, err := s.doChat([]model.ChatMessage{{
		Role:    "user",
		Content: "ping",
	}}, 1)
	return err
}
