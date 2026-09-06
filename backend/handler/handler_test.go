package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"leans/config"
	"leans/model"
	"leans/service"
	"leans/storage"
	"leans/subject"

	"github.com/gin-gonic/gin"
)

// newTestEnv 装配一个带临时存储与空科目库的测试路由。
// AI 默认配置留空，保证 analyze/settings 走"未配置"错误路径，不发真实请求。
func newTestEnv(t *testing.T) (*gin.Engine, *storage.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store, err := storage.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	subjStore, err := subject.NewStore("")
	if err != nil {
		t.Fatalf("subject.NewStore: %v", err)
	}

	cfg := config.Analysis{
		LectureBudget: 6000,
		MaxSections:   2,
		MaxChars:      1500,
		OverviewMax:   1,
		MaxTokens:     4096,
	}
	ai := service.NewAIService(model.AISettings{}, store.GetSettings, cfg.MaxTokens)
	analyzer := service.NewAnalyzer(ai, subjStore, store, cfg)

	r := gin.New()
	api := r.Group("/api")
	{
		sh := NewSubjectHandler(subjStore)
		api.GET("/subjects", sh.List)
		api.GET("/subjects/:id", sh.GetContent)

		ah := NewAnalyzeHandler(analyzer)
		api.POST("/analyze", ah.Handle)
		api.POST("/analyze/stream", ah.HandleStream)

		sth := NewSettingsHandler(store, ai)
		api.GET("/settings", sth.Get)
		api.PUT("/settings", sth.Put)
		api.POST("/settings/active", sth.Active)
		api.POST("/settings/models", sth.Models)
		api.POST("/settings/test", sth.Test)

		hh := NewHistoryHandler(store)
		api.GET("/history", hh.List)
		api.GET("/history/:id", hh.Get)
		api.DELETE("/history", hh.Clear)
	}
	return r, store
}

func doJSON(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", w.Body.String(), err)
	}
	return body.Error
}

func TestSubjectsListEmpty(t *testing.T) {
	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodGet, "/api/subjects", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var items []subject.Summary
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list, got %+v", items)
	}
}

func TestSubjectsListAndGetContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	md := `# 言语理解讲义

## 第一章 中心理解

### 1.1 说理类

内容甲。
`
	if err := os.WriteFile(filepath.Join(dir, "言语理解.md"), []byte(md), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	subjStore, err := subject.NewStore(dir)
	if err != nil {
		t.Fatalf("subject.NewStore: %v", err)
	}
	store, err := storage.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("storage.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	r := gin.New()
	sh := NewSubjectHandler(subjStore)
	r.GET("/api/subjects", sh.List)
	r.GET("/api/subjects/:id", sh.GetContent)

	w := doJSON(t, r, http.MethodGet, "/api/subjects", "")
	var items []subject.Summary
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 || items[0].ID != "言语理解" {
		t.Errorf("list = %+v, want [言语理解]", items)
	}

	w = doJSON(t, r, http.MethodGet, "/api/subjects/言语理解", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get content status = %d, body %s", w.Code, w.Body.String())
	}
	var content struct {
		ID      string            `json:"id"`
		Name    string            `json:"name"`
		Summary string            `json:"summary"`
		Tree    []subject.Section `json:"tree"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &content); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if content.ID != "言语理解" || len(content.Tree) != 1 || content.Tree[0].Title != "第一章 中心理解" {
		t.Errorf("content mismatch: %+v", content)
	}
}

func TestSubjectsGetNotFound(t *testing.T) {
	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodGet, "/api/subjects/nope", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if msg := decodeError(t, w); !strings.Contains(msg, "科目不存在") {
		t.Errorf("error = %q", msg)
	}
}

func TestAnalyzeMissingFields(t *testing.T) {
	r, _ := newTestEnv(t)
	// 缺 subject / content → 400。
	w := doJSON(t, r, http.MethodPost, "/api/analyze", `{"subject":"言语理解"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %s", w.Code, w.Body.String())
	}
}

func TestAnalyzeNotConfigured(t *testing.T) {
	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodPost, "/api/analyze", `{"subject":"言语理解","content":"题目内容"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (AI 未配置), body %s", w.Code, w.Body.String())
	}
	if msg := decodeError(t, w); !strings.Contains(msg, "供应商") {
		t.Errorf("error = %q, want 提示配置 AI 供应商", msg)
	}
}

// TestAnalyzeStreamNotConfigured 流式端点：AI 未配置时应返回 SSE error 事件。
func TestAnalyzeStreamNotConfigured(t *testing.T) {
	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodPost, "/api/analyze/stream", `{"subject":"言语理解","content":"题目内容"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (SSE), body %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "event: error") {
		t.Errorf("SSE body missing error event: %s", body)
	}
	if !strings.Contains(body, "供应商") {
		t.Errorf("SSE error should mention 供应商: %s", body)
	}
}

func TestSettingsGetPut(t *testing.T) {
	r, store := newTestEnv(t)

	w := doJSON(t, r, http.MethodGet, "/api/settings", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, body %s", w.Code, w.Body.String())
	}
	var got model.AISettings
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Providers) != 0 {
		t.Errorf("default providers = %+v, want empty", got.Providers)
	}

	w = doJSON(t, r, http.MethodPut, "/api/settings",
		`{"active_provider_id":"p1","active_model":"deepseek-chat","providers":[{"id":"p1","name":"DeepSeek","base_url":"https://api.deepseek.com/v1","api_key":"sk-x","models":["deepseek-chat"]}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("put status = %d, body %s", w.Code, w.Body.String())
	}

	persisted, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if persisted.ActiveProviderID != "p1" || len(persisted.Providers) != 1 ||
		persisted.Providers[0].APIKey != "sk-x" || persisted.Providers[0].Models[0] != "deepseek-chat" {
		t.Errorf("persisted settings mismatch: %+v", persisted)
	}
}

// TestSettingsActive 切换激活供应商/模型，未知供应商返回 400。
func TestSettingsActive(t *testing.T) {
	r, store := newTestEnv(t)

	in := `{"active_provider_id":"p1","active_model":"m1","providers":[` +
		`{"id":"p1","name":"A","base_url":"https://a/v1","api_key":"k","models":["m1","m2"]},` +
		`{"id":"p2","name":"B","base_url":"https://b/v1","api_key":"k","models":["m9"]}]}`
	if w := doJSON(t, r, http.MethodPut, "/api/settings", in); w.Code != http.StatusOK {
		t.Fatalf("put status = %d, body %s", w.Code, w.Body.String())
	}

	// 切换供应商不传 model → 自动选中该供应商第一个模型。
	w := doJSON(t, r, http.MethodPost, "/api/settings/active", `{"provider_id":"p2"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("active status = %d, body %s", w.Code, w.Body.String())
	}
	persisted, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if persisted.ActiveProviderID != "p2" || persisted.ActiveModel != "m9" {
		t.Errorf("after switch: %+v", persisted)
	}

	// 显式传 model。
	w = doJSON(t, r, http.MethodPost, "/api/settings/active", `{"provider_id":"p2","model":"m9"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("active status = %d, body %s", w.Code, w.Body.String())
	}

	// 未知供应商。
	w = doJSON(t, r, http.MethodPost, "/api/settings/active", `{"provider_id":"nope"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown provider status = %d, want 400", w.Code)
	}
}

// TestSettingsModels 用 mock 上游验证在线获取模型列表。
func TestSettingsModels(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"},{"id":"model-b"}]}`))
	}))
	defer upstream.Close()

	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodPost, "/api/settings/models",
		`{"base_url":"`+upstream.URL+`","api_key":"sk-x"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("models status = %d, body %s", w.Code, w.Body.String())
	}
	var got struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Models) != 2 || got.Models[0] != "model-a" || got.Models[1] != "model-b" {
		t.Errorf("models = %v", got.Models)
	}

	// 缺 base_url → 400。
	w = doJSON(t, r, http.MethodPost, "/api/settings/models", `{"api_key":"k"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing base_url status = %d, want 400", w.Code)
	}
}

func TestSettingsPutInvalidBody(t *testing.T) {
	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodPut, "/api/settings", `{not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSettingsTestNotConfigured(t *testing.T) {
	r, _ := newTestEnv(t)
	w := doJSON(t, r, http.MethodPost, "/api/settings/test", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (未配置), body %s", w.Code, w.Body.String())
	}
	if msg := decodeError(t, w); !strings.Contains(msg, "供应商") {
		t.Errorf("error = %q", msg)
	}
}

func TestHistoryListGetClear(t *testing.T) {
	r, store := newTestEnv(t)

	// 初始为空。
	w := doJSON(t, r, http.MethodGet, "/api/history", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", w.Code, w.Body.String())
	}
	var items []model.HistoryItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty history, got %+v", items)
	}

	// 写入一条后列表可见（不含 result）。
	if _, err := store.AddHistory(model.HistoryItem{
		Subject:      "言语理解",
		Question:     "题目一",
		Category:     "中心理解",
		Result:       `{"category":"中心理解"}`,
		Model:        "test-model",
		Tokens:       1234,
		ElapsedMS:    5600,
		FirstTokenMS: 900,
	}); err != nil {
		t.Fatalf("AddHistory: %v", err)
	}
	w = doJSON(t, r, http.MethodGet, "/api/history", "")
	items = nil
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 || items[0].Question != "题目一" || items[0].Result != "" {
		t.Errorf("list mismatch: %+v", items)
	}
	if items[0].Tokens != 1234 || items[0].ElapsedMS != 5600 {
		t.Errorf("tokens/elapsed not listed: %+v", items[0])
	}
	if items[0].Model != "test-model" || items[0].FirstTokenMS != 900 {
		t.Errorf("model/first_token not listed: %+v", items[0])
	}

	// 单条获取带回 result。
	w = doJSON(t, r, http.MethodGet, "/api/history/1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, body %s", w.Code, w.Body.String())
	}
	var item model.HistoryItem
	if err := json.Unmarshal(w.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if item.Result != `{"category":"中心理解"}` {
		t.Errorf("result = %q", item.Result)
	}

	// 不存在的记录 → 404。
	w = doJSON(t, r, http.MethodGet, "/api/history/999", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("get missing status = %d, want 404", w.Code)
	}

	// 非法 id → 400。
	w = doJSON(t, r, http.MethodGet, "/api/history/abc", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get invalid id status = %d, want 400", w.Code)
	}

	// 清空。
	w = doJSON(t, r, http.MethodDelete, "/api/history", "")
	if w.Code != http.StatusOK {
		t.Fatalf("clear status = %d, body %s", w.Code, w.Body.String())
	}
	items, _ = store.ListHistory(10)
	if len(items) != 0 {
		t.Errorf("history not cleared: %+v", items)
	}
}
