package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cn-maul/rosetta"
	"leans/model"
)

// aiTimeout 限制单次 AI 调用（含流式全程）的最长耗时。
const aiTimeout = 5 * time.Minute

type SettingsProvider func() (model.AISettings, error)

// AIService 通过 rosetta 统一接入 AI 服务：协议细节（OpenAI 兼容字段
// 探测降级、SSE 解析、传输重试）由 rosetta 处理，本类型负责多供应商
// 设置解析与应用层错误翻译。
type AIService struct {
	defaults  model.AISettings
	settings  SettingsProvider
	maxTokens int

	mu        sync.Mutex
	client    *rosetta.Client
	clientKey string
}

func NewAIService(defaults model.AISettings, provider SettingsProvider, maxTokens int) *AIService {
	return &AIService{
		defaults:  defaults,
		settings:  provider,
		maxTokens: maxTokens,
	}
}

// Settings 返回归一化后的设置：有持久化供应商时以持久化为准，否则用
// 默认值（来自 config.yaml 的种子供应商）。
func (s *AIService) Settings() (model.AISettings, error) {
	set := s.defaults
	if s.settings != nil {
		persisted, err := s.settings()
		if err != nil {
			return set, err
		}
		if len(persisted.Providers) > 0 {
			set = persisted
		}
	}
	return NormalizeSettings(set), nil
}

// NormalizeSettings 修复设置的自洽性：补齐/去重供应商 ID，清理空模型，
// 并把激活供应商/模型修正为有效值。所有读写入口统一经过它。
func NormalizeSettings(set model.AISettings) model.AISettings {
	seen := map[string]bool{}
	for i := range set.Providers {
		p := &set.Providers[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			p.Name = fmt.Sprintf("供应商 %d", i+1)
		}
		p.BaseURL = strings.TrimSpace(p.BaseURL)
		p.Protocol = resolveProtocol(p.Protocol)
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" || seen[p.ID] {
			base := p.ID
			if base == "" {
				base = "provider"
			}
			for n := 2; ; n++ {
				candidate := base
				if n > 2 || seen[base] {
					candidate = fmt.Sprintf("%s-%d", base, n)
				}
				if !seen[candidate] {
					p.ID = candidate
					break
				}
			}
		}
		seen[p.ID] = true

		models := make([]string, 0, len(p.Models))
		modelSeen := map[string]bool{}
		for _, m := range p.Models {
			m = strings.TrimSpace(m)
			if m != "" && !modelSeen[m] {
				models = append(models, m)
				modelSeen[m] = true
			}
		}
		p.Models = models
	}

	if set.ActiveProviderID != "" {
		_, ok := findProvider(set, set.ActiveProviderID)
		if !ok {
			set.ActiveProviderID = ""
		}
	}
	if set.ActiveProviderID == "" && len(set.Providers) > 0 {
		set.ActiveProviderID = set.Providers[0].ID
	}
	if p, ok := findProvider(set, set.ActiveProviderID); ok {
		if !containsModel(p.Models, set.ActiveModel) {
			set.ActiveModel = ""
			if len(p.Models) > 0 {
				set.ActiveModel = p.Models[0]
			}
		}
	} else {
		set.ActiveModel = ""
	}
	return set
}

// resolveProtocol 把协议字段规整为 rosetta 可用的取值：auto/空 会在
// buildClient 时主动探测端点。
func resolveProtocol(p string) string {
	if !model.ValidProtocol(p) {
		return model.ProtocolAuto
	}
	if p == "" {
		return model.ProtocolAuto
	}
	return p
}

func findProvider(set model.AISettings, id string) (model.AIProvider, bool) {
	for _, p := range set.Providers {
		if p.ID == id {
			return p, true
		}
	}
	return model.AIProvider{}, false
}

func containsModel(models []string, m string) bool {
	for _, x := range models {
		if x == m {
			return true
		}
	}
	return false
}

// Resolved 是一次调用实际使用的配置：激活供应商 + 激活模型。
type Resolved struct {
	Provider model.AIProvider
	Model    string
}

// Resolve 解析当前应使用的供应商与模型，配置不完整时返回
// ErrNotConfigured（handler 映射为 4xx）。
func (s *AIService) Resolve() (Resolved, error) {
	set, err := s.Settings()
	if err != nil {
		return Resolved{}, fmt.Errorf("读取AI配置失败: %w", err)
	}
	p, ok := findProvider(set, set.ActiveProviderID)
	if !ok {
		return Resolved{}, fmt.Errorf("%w: 未选择 AI 供应商，请先在设置中添加", ErrNotConfigured)
	}
	if p.APIKey == "" {
		return Resolved{}, fmt.Errorf("%w: 未配置 API Key，请先在设置中填写", ErrNotConfigured)
	}
	if p.BaseURL == "" {
		return Resolved{}, fmt.Errorf("%w: 未配置 API Base URL", ErrNotConfigured)
	}
	if set.ActiveModel == "" {
		return Resolved{}, fmt.Errorf("%w: 未选择模型，请先在设置中添加或获取", ErrNotConfigured)
	}
	return Resolved{Provider: p, Model: set.ActiveModel}, nil
}

// Validate ensures the resolved settings can be used.
func (s *AIService) Validate() error {
	_, err := s.Resolve()
	return err
}

// clientFor 返回与当前供应商配置匹配的 rosetta 客户端。设置可在运行
// 时修改，因此只缓存最近一个实例：配置未变时复用（保留 rosetta 对不
// 兼容服务的探测降级状态，auto 协议的探测结果也随缓存固定），变了则
// 重建。
func (s *AIService) clientFor(ctx context.Context, p model.AIProvider) (*rosetta.Client, error) {
	spec := resolveProtocol(p.Protocol)
	key := p.BaseURL + "\x00" + p.APIKey + "\x00" + spec
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil && s.clientKey == key {
		return s.client, nil
	}
	c, err := buildClient(ctx, p.BaseURL, p.APIKey, spec, aiTimeout)
	if err != nil {
		return nil, err
	}
	s.client, s.clientKey = c, key
	return c, nil
}

// buildClient 按 endpoint+凭据+协议构造 rosetta 客户端。协议为 auto 时
// 通过 GET /models 主动探测（Bearer 与 x-api-key 两种鉴权都尝试），
// 探测失败回落 OpenAI Chat 兼容。
func buildClient(ctx context.Context, baseURL, apiKey, protocol string, timeout time.Duration) (*rosetta.Client, error) {
	opts := []rosetta.Option{
		rosetta.WithEndpoint(strings.TrimSpace(baseURL)),
		rosetta.WithAPIKey(strings.TrimSpace(apiKey)),
		rosetta.WithTimeout(timeout),
	}
	if protocol == model.ProtocolAuto {
		detectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if detected, err := rosetta.DetectProtocol(detectCtx, baseURL, apiKey); err == nil {
			opts = append(opts, rosetta.WithProtocol(detected))
		} else {
			opts = append(opts, rosetta.WithProtocol(rosetta.ProtoOpenAIChat))
		}
	} else {
		opts = append(opts, rosetta.WithProtocol(rosetta.Protocol(protocol)))
	}
	return rosetta.NewClient(opts...)
}

// toRosettaMessages 把应用层消息转成 rosetta 统一消息。
func toRosettaMessages(messages []model.ChatMessage) []rosetta.Message {
	out := make([]rosetta.Message, 0, len(messages))
	for _, m := range messages {
		out = append(out, rosetta.Message{
			Role:   rosetta.Role(m.Role),
			Blocks: []rosetta.Block{{Type: rosetta.BlockText, Text: m.Content}},
		})
	}
	return out
}

// toModelUsage 转换 token 统计；上游未返回用量时保持 nil 语义。
func toModelUsage(u rosetta.Usage) *model.Usage {
	if u.IsZero() {
		return nil
	}
	return &model.Usage{
		PromptTokens:     int(u.InputTokens),
		CompletionTokens: int(u.OutputTokens),
		TotalTokens:      int(u.TotalTokens),
	}
}

// translateErr 把 rosetta 错误翻译成前端可读的消息，格式与旧实现一致。
func translateErr(err error) error {
	switch {
	case errors.Is(err, rosetta.ErrNoAPIKey):
		return fmt.Errorf("%w: 未配置 API Key，请先在设置中填写", ErrNotConfigured)
	case errors.Is(err, rosetta.ErrNoEndpoint):
		return fmt.Errorf("%w: 未配置 API Base URL", ErrNotConfigured)
	}
	var apiErr *rosetta.APIError
	if errors.As(err, &apiErr) {
		msg := apiErr.Message
		if msg == "" {
			msg = http.StatusText(apiErr.StatusCode)
		}
		return fmt.Errorf("API 错误 (status %d): %s", apiErr.StatusCode, msg)
	}
	var transErr *rosetta.TransportError
	if errors.As(err, &transErr) {
		return fmt.Errorf("发送请求失败: %v", transErr.Err)
	}
	return err
}

func (s *AIService) ChatCompletion(ctx context.Context, messages []model.ChatMessage) (string, *model.Usage, error) {
	resolved, err := s.Resolve()
	if err != nil {
		return "", nil, err
	}
	client, err := s.clientFor(ctx, resolved.Provider)
	if err != nil {
		return "", nil, translateErr(err)
	}
	ctx, cancel := context.WithTimeout(ctx, aiTimeout)
	defer cancel()
	resp, err := client.Chat(ctx, &rosetta.ChatRequest{
		Model:           resolved.Model,
		Messages:        toRosettaMessages(messages),
		MaxOutputTokens: s.maxTokens,
	})
	if err != nil {
		return "", nil, translateErr(err)
	}
	return resp.Text(), toModelUsage(resp.Usage), nil
}

// ChatCompletionStream 流式调用，每收到一段内容增量就调用一次 onDelta。
// ctx 贯穿整个流：调用方（如 HTTP 请求）被取消时流随之终止。
// 返回完整内容、usage 统计，以及从请求发出到首个内容增量的耗时
// （首字响应时间，毫秒）。
func (s *AIService) ChatCompletionStream(ctx context.Context, messages []model.ChatMessage, onDelta func(string)) (string, *model.Usage, int64, error) {
	resolved, err := s.Resolve()
	if err != nil {
		return "", nil, 0, err
	}
	client, err := s.clientFor(ctx, resolved.Provider)
	if err != nil {
		return "", nil, 0, translateErr(err)
	}
	ctx, cancel := context.WithTimeout(ctx, aiTimeout)
	defer cancel()

	start := time.Now()
	stream, err := client.ChatStream(ctx, &rosetta.ChatRequest{
		Model:           resolved.Model,
		Messages:        toRosettaMessages(messages),
		MaxOutputTokens: s.maxTokens,
	})
	if err != nil {
		return "", nil, 0, translateErr(err)
	}
	defer stream.Close()

	var content strings.Builder
	var firstTokenMS int64
	for stream.Next() {
		ev := stream.Event()
		if ev.Type != rosetta.EventTextDelta {
			continue
		}
		if firstTokenMS == 0 {
			firstTokenMS = time.Since(start).Milliseconds()
		}
		content.WriteString(ev.Text)
		if onDelta != nil {
			onDelta(ev.Text)
		}
	}
	if err := stream.Err(); err != nil {
		return content.String(), nil, firstTokenMS, translateErr(err)
	}

	usage := toModelUsage(stream.Usage())
	if content.Len() == 0 {
		return "", usage, firstTokenMS, fmt.Errorf("流式响应中没有内容")
	}
	return content.String(), usage, firstTokenMS, nil
}

// FetchModels 从服务端拉取可用模型 ID 列表（GET /models），供设置
// 界面"获取模型"使用。凭据来自表单，不依赖已保存的设置；API Key 为空
// 时用占位符兼容 Ollama/vLLM 等无需鉴权的本地服务。
func (s *AIService) FetchModels(ctx context.Context, baseURL, apiKey, protocol string) ([]string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("%w: 未配置 API Base URL", ErrNotConfigured)
	}
	if strings.TrimSpace(apiKey) == "" {
		apiKey = "EMPTY"
	}
	client, err := buildClient(ctx, baseURL, apiKey, resolveProtocol(protocol), 15*time.Second)
	if err != nil {
		return nil, translateErr(err)
	}
	infos, err := client.ListModels(ctx)
	if err != nil {
		return nil, translateErr(err)
	}
	out := make([]string, 0, len(infos))
	seen := map[string]bool{}
	for _, info := range infos {
		if info.ID != "" && !seen[info.ID] {
			out = append(out, info.ID)
			seen[info.ID] = true
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("服务端未返回任何模型，请确认 Base URL 是否正确")
	}
	return out, nil
}

// Ping 用当前激活的配置发一个最小请求，验证连通性。返回 nil 表示成功。
func (s *AIService) Ping() error {
	resolved, err := s.Resolve()
	if err != nil {
		return err
	}
	return s.PingProvider(resolved.Provider, resolved.Model)
}

// PingProvider 用给定供应商配置与模型发一个最小请求，验证连通性，
// 供设置界面在保存前测试。model 为空时使用该供应商的第一个模型。
func (s *AIService) PingProvider(p model.AIProvider, modelName string) error {
	if p.BaseURL == "" {
		return fmt.Errorf("%w: 未配置 API Base URL", ErrNotConfigured)
	}
	if p.APIKey == "" {
		return fmt.Errorf("%w: 未配置 API Key，请先在设置中填写", ErrNotConfigured)
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		if len(p.Models) == 0 {
			return fmt.Errorf("%w: 请先添加或获取模型后再测试", ErrNotConfigured)
		}
		modelName = p.Models[0]
	}
	client, err := buildClient(context.Background(), p.BaseURL, p.APIKey, resolveProtocol(p.Protocol), 30*time.Second)
	if err != nil {
		return translateErr(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = client.Chat(ctx, &rosetta.ChatRequest{
		Model:           modelName,
		Messages:        []rosetta.Message{rosetta.User("ping")},
		MaxOutputTokens: 1,
	})
	return translateErr(err)
}
