package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"leans/config"
	"leans/subject"
)

// newTestAnalyzer 装配一个指向 mock 服务端的 Analyzer，科目库只含一个临时科目。
func newTestAnalyzer(t *testing.T, ai *AIService, subjectBody string) (*Analyzer, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "测试科目.md"), []byte(subjectBody), 0o644); err != nil {
		t.Fatalf("write subject: %v", err)
	}
	subjStore, err := subject.NewStore(dir)
	if err != nil {
		t.Fatalf("subject.NewStore: %v", err)
	}
	if len(subjStore.List()) == 0 {
		t.Fatal("subject store is empty")
	}
	return NewAnalyzer(ai, subjStore, nil, config.Analysis{
		LectureBudget: 6000,
		MaxSections:   2,
		MaxChars:      1500,
		OverviewMax:   1,
		MaxTokens:     4096,
	}), "测试科目"
}

const testSubjectBody = "# 测试科目\n\n## 第一章 数量关系\n\n### 1.1 工程问题\n\n总量设为 1，效率 = 1 / 时间；合作取效率和，交替工作按周期计算剩余量。\n"

// asContentString 把一段 JSON 文本包成 JSON 字符串字面量，用于塞进
// OpenAI 响应的 message.content —— 直接内联会变成对象，宽松解码器会置空。
func asContentString(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	quoted, err := json.Marshal(string(raw))
	if err != nil {
		t.Fatalf("marshal content string: %v", err)
	}
	return quoted
}

// 上游流式发了一半就断（缺终止事件 → rosetta 报截断），非流式则正常返回。
// 截断时已推给前端的增量不能被整条丢弃：应走「解析失败 → 非流式重试」并成功。
func TestAnalyzeStreamTruncatedKeepsPartialAndRetries(t *testing.T) {
	answer := asContentString(t, map[string]any{"category": "数量关系", "answer": "12 天"})

	var unaryCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if bytes.Contains(body, []byte(`"stream":true`)) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"id\":\"1\",\"model\":\"test-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"{\\\"category\\\":\"}}]}\n\n")
			w.(http.Flusher).Flush()
			return // 缺 finish_reason / [DONE] → 截断
		}
		unaryCalls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"1","model":"test-model","choices":[{"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`, answer)
	}))
	t.Cleanup(srv.Close)

	analyzer, subjectID := newTestAnalyzer(t, newTestService(t, srv.URL), testSubjectBody)

	var deltas, statuses []string
	result, err := analyzer.AnalyzeStream(context.Background(), AnalyzeOption{
		Subject: subjectID,
		Content: "某工程甲单独做需 20 天，乙单独做需 30 天，合作需多少天？",
	}, StreamCallbacks{
		OnDelta:  func(text string) { deltas = append(deltas, text) },
		OnStatus: func(msg string) { statuses = append(statuses, msg) },
	})
	if err != nil {
		t.Fatalf("AnalyzeStream: %v", err)
	}
	if result.Category != "数量关系" || result.Answer != "12 天" {
		t.Errorf("result = %+v", result)
	}
	// 截断前收到的增量已经推给前端了，重试不应把它抹掉。
	if got := strings.Join(deltas, ""); got != `{"category":` {
		t.Errorf("deltas = %q, want partial content kept", got)
	}
	if len(statuses) == 0 {
		t.Error("want a status notify for the unary fallback")
	}
	if unaryCalls != 1 {
		t.Errorf("unaryCalls = %d, want 1", unaryCalls)
	}
}

// 一个字都没收到时必须立刻失败：不进入非流式重试，避免白花一次调用。
func TestAnalyzeStreamFailsFastWhenNothingReceived(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	analyzer, subjectID := newTestAnalyzer(t, newTestService(t, srv.URL), testSubjectBody)

	_, err := analyzer.AnalyzeStream(context.Background(),
		AnalyzeOption{Subject: subjectID, Content: "题目"}, StreamCallbacks{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "AI 调用失败") {
		t.Errorf("err = %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1（空响应不重试）", calls)
	}
}

// 非流式空回复重试一次：第一次返回空正文，第二次给出正文。
func TestAnalyzeRetriesEmptyCompletionOnce(t *testing.T) {
	answer := asContentString(t, map[string]any{"category": "判断推理", "answer": "B"})
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			fmt.Fprint(w, `{"id":"1","model":"test-model","choices":[{"message":{"role":"assistant","content":""},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":0,"total_tokens":10}}`)
			return
		}
		fmt.Fprintf(w, `{"id":"2","model":"test-model","choices":[{"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`, answer)
	}))
	t.Cleanup(srv.Close)

	analyzer, subjectID := newTestAnalyzer(t, newTestService(t, srv.URL), testSubjectBody)

	result, err := analyzer.Analyze(context.Background(),
		AnalyzeOption{Subject: subjectID, Content: "题目"})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if result.Category != "判断推理" || result.Answer != "B" {
		t.Errorf("result = %+v", result)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2（空回复重试一次）", calls)
	}
	if result.Meta.TotalTokens != 15 {
		t.Errorf("tokens = %d, want 15（应记录重试那次的 usage）", result.Meta.TotalTokens)
	}
}
