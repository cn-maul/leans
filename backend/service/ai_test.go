package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"leans/model"
)

// newTestService 构造指向 mock 服务端的 AIService（配置走 defaults）。
func newTestService(t *testing.T, baseURL string) *AIService {
	t.Helper()
	return NewAIService(model.AISettings{
		ActiveProviderID: "p1",
		ActiveModel:      "test-model",
		Providers: []model.AIProvider{{
			ID:       "p1",
			Name:     "Test",
			BaseURL:  baseURL,
			APIKey:   "test-key",
			Protocol: model.ProtocolOpenAIChat,
			Models:   []string{"test-model"},
		}},
	}, nil, 1024)
}

func TestChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("unexpected authorization: %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"1","model":"test-model","choices":[{"message":{"role":"assistant","content":"你好"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer srv.Close()

	svc := newTestService(t, srv.URL)
	content, usage, err := svc.ChatCompletion(context.Background(), []model.ChatMessage{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if content != "你好" {
		t.Errorf("content = %q", content)
	}
	if usage == nil || usage.PromptTokens != 10 || usage.CompletionTokens != 5 || usage.TotalTokens != 15 {
		t.Errorf("usage = %+v", usage)
	}
}

func TestChatCompletionStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunk := func(payload string) {
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
		chunk(`{"id":"1","model":"test-model","choices":[{"index":0,"delta":{"role":"assistant"}}]}`)
		time.Sleep(10 * time.Millisecond) // 保证首字耗时可测
		chunk(`{"choices":[{"index":0,"delta":{"content":"你"}}]}`)
		chunk(`{"choices":[{"index":0,"delta":{"content":"好"}}]}`)
		chunk(`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	svc := newTestService(t, srv.URL)
	var deltas []string
	content, usage, firstTokenMS, err := svc.ChatCompletionStream(
		context.Background(),
		[]model.ChatMessage{{Role: "user", Content: "hi"}},
		func(text string) { deltas = append(deltas, text) },
	)
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	if content != "你好" {
		t.Errorf("content = %q", content)
	}
	if strings.Join(deltas, "") != "你好" {
		t.Errorf("deltas = %q", deltas)
	}
	if usage == nil || usage.TotalTokens != 12 {
		t.Errorf("usage = %+v", usage)
	}
	if firstTokenMS <= 0 {
		t.Errorf("firstTokenMS = %d", firstTokenMS)
	}
}

func TestChatCompletionStreamNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	svc := newTestService(t, srv.URL)
	_, _, _, err := svc.ChatCompletionStream(context.Background(), []model.ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "流式响应中没有内容") {
		t.Fatalf("err = %v", err)
	}
}

func TestTranslateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Invalid API key","type":"auth"}}`)
	}))
	defer srv.Close()

	svc := newTestService(t, srv.URL)
	err := svc.Ping()
	if err == nil || !strings.Contains(err.Error(), "API 错误 (status 401)") || !strings.Contains(err.Error(), "Invalid API key") {
		t.Fatalf("Ping err = %v", err)
	}
}

func TestNotConfigured(t *testing.T) {
	svc := NewAIService(model.AISettings{}, nil, 1024)
	if _, _, err := svc.ChatCompletion(context.Background(), []model.ChatMessage{{Role: "user", Content: "hi"}}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	if err := svc.Ping(); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Ping err = %v", err)
	}
}

func TestNormalizeSettings(t *testing.T) {
	set := model.AISettings{
		ActiveProviderID: "missing",
		ActiveModel:      "m2",
		Providers: []model.AIProvider{
			{ID: "p1", Name: "  ", BaseURL: " https://a.com/v1 ", Models: []string{"", "m1", "m1", " m2 "}},
			{BaseURL: "https://b.com/v1", Models: []string{"m9"}},
		},
	}
	got := NormalizeSettings(set)
	if len(got.Providers) != 2 {
		t.Fatalf("providers = %+v", got.Providers)
	}
	p0, p1 := got.Providers[0], got.Providers[1]
	if p0.ID != "p1" || p0.Name == "" || p0.Models[0] != "m1" || len(p0.Models) != 2 {
		t.Errorf("p0 = %+v", p0)
	}
	if p1.ID == "" || p1.ID == p0.ID || p1.Name == "" {
		t.Errorf("p1 = %+v", p1)
	}
	// 激活供应商失效回落到第一个供应商；原激活模型仍有效则保留。
	if got.ActiveProviderID != p0.ID || got.ActiveModel != "m2" {
		t.Errorf("active = %q/%q", got.ActiveProviderID, got.ActiveModel)
	}

	// 激活模型完全无效时回落到第一个模型。
	set2 := model.AISettings{
		ActiveProviderID: "p1",
		ActiveModel:      "nonexistent",
		Providers:        []model.AIProvider{{ID: "p1", Name: "A", Models: []string{"m1", "m2"}}},
	}
	got2 := NormalizeSettings(set2)
	if got2.ActiveModel != "m1" {
		t.Errorf("fallback model = %q, want m1", got2.ActiveModel)
	}

	// 空设置保持为空，不产生幽灵供应商。
	empty := NormalizeSettings(model.AISettings{})
	if len(empty.Providers) != 0 || empty.ActiveProviderID != "" {
		t.Errorf("empty = %+v", empty)
	}
}

func TestFetchModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"model-b"},{"id":"model-a"},{"id":"model-b"},{"id":""}]}`)
	}))
	defer srv.Close()

	svc := NewAIService(model.AISettings{}, nil, 1024)
	models, err := svc.FetchModels(context.Background(), srv.URL, "test-key", model.ProtocolOpenAIChat)
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(models) != 2 || models[0] != "model-a" || models[1] != "model-b" {
		t.Errorf("models = %v", models)
	}

	// 协议 auto 时探测失败应回落 OpenAI 兼容，仍能拉到列表。
	models, err = svc.FetchModels(context.Background(), srv.URL, "", model.ProtocolAuto)
	if err != nil {
		t.Fatalf("FetchModels(auto): %v", err)
	}
	if len(models) != 2 {
		t.Errorf("auto models = %v", models)
	}

	if _, err := svc.FetchModels(context.Background(), "", "k", ""); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("empty base url err = %v", err)
	}
}
