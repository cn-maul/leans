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

// sseServer 起一个 mock 服务端，body 里按序写 SSE data 行；返回的 write 供
// 每个用例自定义流内容。
func sseServer(t *testing.T, write func(w http.ResponseWriter)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		write(w)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func sseChunk(t *testing.T, w http.ResponseWriter, payload string) {
	t.Helper()
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		t.Errorf("write chunk: %v", err)
	}
	w.(http.Flusher).Flush()
}

// 上游发了一半就断（没有 finish_reason、没有 [DONE]）：rosetta 报
// ErrStreamTruncated，翻译层必须给出中文可读文案，且已收到的增量不能被丢掉
// —— AnalyzeStream 依赖它走「解析失败 → 非流式重试」，而不是白花一次 token。
func TestChatCompletionStreamTruncatedKeepsPartial(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter) {
		sseChunk(t, w, `{"id":"1","model":"test-model","choices":[{"index":0,"delta":{"content":"{\"category\":"}}]}`)
		// 直接返回：连接正常关闭但缺终止事件 → 截断。
	})

	svc := newTestService(t, srv.URL)
	content, _, _, err := svc.ChatCompletionStream(context.Background(),
		[]model.ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("want truncation error, got nil")
	}
	if content != `{"category":` {
		t.Errorf("content = %q, want partial content preserved", content)
	}
	if !strings.Contains(err.Error(), "上游连接中断") {
		t.Errorf("err = %v, want 中文翻译", err)
	}
	if strings.Contains(err.Error(), "rosetta:") || strings.Contains(err.Error(), "truncated") {
		t.Errorf("err = %v, should not leak raw SDK error", err)
	}
}

// 思考型模型把 max_tokens 烧在思考期：正文为空但思考非空。这条报错本身
// 是如实的，关键是要把它和「上游真空」区分开，并给出可诊断的字段。
func TestChatCompletionStreamThinkingOnly(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter) {
		// 首字耗时靠真实时间差测量，抖一下避免毫秒级截断成 0。
		time.Sleep(10 * time.Millisecond)
		sseChunk(t, w, `{"id":"1","model":"test-model","choices":[{"index":0,"delta":{"reasoning_content":"先看清题意"}}]}`)
		sseChunk(t, w, `{"choices":[{"index":0,"delta":{"reasoning_content":"再逐项验证"}}]}`)
		sseChunk(t, w, `{"choices":[{"index":0,"delta":{},"finish_reason":"length"}],"usage":{"prompt_tokens":100,"completion_tokens":4096,"total_tokens":4196}}`)
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	})

	svc := newTestService(t, srv.URL)
	content, usage, firstTokenMS, err := svc.ChatCompletionStream(context.Background(),
		[]model.ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if content != "" {
		t.Errorf("content = %q, want empty", content)
	}
	if !errors.Is(err, ErrEmptyCompletion) {
		t.Fatalf("err = %v, want ErrEmptyCompletion", err)
	}
	for _, want := range []string{"只输出了思考内容", "stop=length"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want contains %q", err, want)
		}
	}
	// 思考增量同样算「上游开始回应」，否则首字耗时会被误报成 0。
	if firstTokenMS <= 0 {
		t.Errorf("firstTokenMS = %d, want > 0", firstTokenMS)
	}
	if usage == nil || usage.CompletionTokens != 4096 {
		t.Errorf("usage = %+v, want completion=4096", usage)
	}
}

// 上游给了终止事件却一个增量都没写（内容过滤等合法空回复）：此时 stop
// 是仅有的线索，必须出现在文案里。
func TestChatCompletionStreamEmptyWithStopReason(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter) {
		sseChunk(t, w, `{"choices":[{"index":0,"delta":{},"finish_reason":"content_filter"}],"usage":{"prompt_tokens":10,"completion_tokens":0,"total_tokens":10}}`)
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	})

	svc := newTestService(t, srv.URL)
	_, _, _, err := svc.ChatCompletionStream(context.Background(),
		[]model.ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if !errors.Is(err, ErrEmptyCompletion) {
		t.Fatalf("err = %v, want ErrEmptyCompletion", err)
	}
	if !strings.Contains(err.Error(), "stop=content_filter") {
		t.Errorf("err = %v, want stop reason included", err)
	}
}

// 非流式同样要能认出「只有思考」。
func TestChatCompletionThinkingOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"1","model":"test-model","choices":[{"message":{"role":"assistant","content":"","reasoning_content":"想了很久"},"finish_reason":"length"}],"usage":{"prompt_tokens":100,"completion_tokens":4096,"total_tokens":4196}}`)
	}))
	t.Cleanup(srv.Close)

	svc := newTestService(t, srv.URL)
	content, usage, err := svc.ChatCompletion(context.Background(),
		[]model.ChatMessage{{Role: "user", Content: "hi"}})
	if content != "" {
		t.Errorf("content = %q, want empty", content)
	}
	if !errors.Is(err, ErrEmptyCompletion) {
		t.Fatalf("err = %v, want ErrEmptyCompletion", err)
	}
	if !strings.Contains(err.Error(), "只输出了思考内容") || !strings.Contains(err.Error(), "stop=length") {
		t.Errorf("err = %v, want thinking-only diagnosis", err)
	}
	if usage == nil || usage.TotalTokens != 4196 {
		t.Errorf("usage = %+v, want total=4196", usage)
	}
}
