package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"leans/model"
	"net/http"
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
		return fmt.Errorf("未配置 API Key，请先在设置中填写")
	}
	if set.BaseURL == "" {
		return fmt.Errorf("未配置 API Base URL")
	}
	if set.Model == "" {
		return fmt.Errorf("未配置模型名称")
	}
	return nil
}

func (s *AIService) ChatCompletion(messages []model.ChatMessage) (string, *model.Usage, error) {
	set, err := s.Settings()
	if err != nil {
		return "", nil, fmt.Errorf("读取AI配置失败: %w", err)
	}
	if set.APIKey == "" {
		return "", nil, fmt.Errorf("未配置 API Key，请先在设置中填写")
	}

	reqBody := model.ChatRequest{
		Model:     set.Model,
		Messages:  messages,
		MaxTokens: s.maxTokens,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("marshal request: %w", err)
	}

	url := set.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+set.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
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

// Ping 用当前配置发一个最小请求，验证 API Key / Base URL / 模型连通性。
// 返回 nil 表示连接成功。
func (s *AIService) Ping() error {
	set, err := s.Settings()
	if err != nil {
		return err
	}
	if set.APIKey == "" {
		return fmt.Errorf("未配置 API Key")
	}
	if set.BaseURL == "" {
		return fmt.Errorf("未配置 API Base URL")
	}
	if set.Model == "" {
		return fmt.Errorf("未配置模型名称")
	}

	reqBody := model.ChatRequest{
		Model: set.Model,
		Messages: []model.ChatMessage{{
			Role:    "user",
			Content: "ping",
		}},
		MaxTokens: 1,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := set.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+set.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 错误 (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}