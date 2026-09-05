package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"leans/model"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func modelAISettings() model.AISettings {
	return model.AISettings{
		Provider: "deepseek",
		APIKey:   "sk-test",
		BaseURL:  "https://api.deepseek.com/v1",
		Model:    "deepseek-chat",
	}
}

func openRaw(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDBFileCreated(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()
	if _, err := os.Stat(filepath.Join(dir, "leans.db")); err != nil {
		t.Errorf("db file not created: %v", err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.Provider != "" || got.APIKey != "" {
		t.Errorf("empty settings expected, got %+v", got)
	}

	in := modelAISettings()
	if err := s.SaveSettings(in); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err = s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after save: %v", err)
	}
	if got.Provider != "deepseek" || got.Model != "deepseek-chat" ||
		got.BaseURL != "https://api.deepseek.com/v1" || got.APIKey != "sk-test" {
		t.Errorf("settings mismatch: %+v", got)
	}
}

func TestHistoryCRUD(t *testing.T) {
	s := newTestStore(t)

	id, err := s.AddHistory(model.HistoryItem{
		Subject:      "言语理解",
		Question:     "题目一",
		Category:     "中心理解",
		Result:       `{"category":"中心理解"}`,
		Tokens:       1234,
		ElapsedMS:    5600,
		FirstTokenMS: 900,
	})
	if err != nil {
		t.Fatalf("AddHistory: %v", err)
	}
	if id == 0 {
		t.Error("AddHistory returned id 0")
	}

	items, err := s.ListHistory(10)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(items) != 1 || items[0].Question != "题目一" || items[0].Category != "中心理解" {
		t.Errorf("list mismatch: %+v", items)
	}
	if items[0].Tokens != 1234 || items[0].ElapsedMS != 5600 {
		t.Errorf("tokens/elapsed mismatch: %+v", items[0])
	}
	if items[0].Result != "" {
		t.Errorf("list should omit result, got %q", items[0].Result)
	}

	got, err := s.GetHistory(id)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if got.Result != `{"category":"中心理解"}` {
		t.Errorf("result mismatch: %q", got.Result)
	}

	if err := s.ClearHistory(); err != nil {
		t.Fatalf("ClearHistory: %v", err)
	}
	items, _ = s.ListHistory(10)
	if len(items) != 0 {
		t.Errorf("history not cleared: %d items", len(items))
	}
}

func TestHistoryLimit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 5; i++ {
		if _, err := s.AddHistory(model.HistoryItem{Subject: "s", Question: "q", Category: "c", Result: "{}"}); err != nil {
			t.Fatalf("AddHistory: %v", err)
		}
	}
	items, err := s.ListHistory(3)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("limit not applied: %d items", len(items))
	}
}

func TestStats(t *testing.T) {
	s := newTestStore(t)

	// 空库时统计应为零值。
	st, err := s.GetStats()
	if err != nil {
		t.Fatalf("GetStats empty: %v", err)
	}
	if st.TotalQuestions != 0 || st.TotalTokens != 0 || st.AvgTokens != 0 {
		t.Errorf("empty stats mismatch: %+v", st)
	}

	if _, err := s.AddHistory(model.HistoryItem{Subject: "s", Question: "q1", Category: "c", Result: "{}", Tokens: 1000, ElapsedMS: 5000, FirstTokenMS: 800}); err != nil {
		t.Fatalf("AddHistory: %v", err)
	}
	if _, err := s.AddHistory(model.HistoryItem{Subject: "s", Question: "q2", Category: "c", Result: "{}", Tokens: 2000, ElapsedMS: 7000, FirstTokenMS: 1600}); err != nil {
		t.Fatalf("AddHistory: %v", err)
	}

	st, err = s.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if st.TotalQuestions != 2 {
		t.Errorf("total_questions = %d, want 2", st.TotalQuestions)
	}
	if st.TotalTokens != 3000 {
		t.Errorf("total_tokens = %d, want 3000", st.TotalTokens)
	}
	if st.AvgTokens != 1500 {
		t.Errorf("avg_tokens = %v, want 1500", st.AvgTokens)
	}
	if st.AvgFirstTokenMS != 1200 {
		t.Errorf("avg_first_token_ms = %v, want 1200", st.AvgFirstTokenMS)
	}
	if st.TotalElapsedMS != 12000 {
		t.Errorf("total_elapsed_ms = %d, want 12000", st.TotalElapsedMS)
	}

	// 清空后统计归零。
	if err := s.ClearHistory(); err != nil {
		t.Fatalf("ClearHistory: %v", err)
	}
	st, _ = s.GetStats()
	if st.TotalQuestions != 0 || st.TotalTokens != 0 {
		t.Errorf("stats after clear: %+v", st)
	}
}

func TestMigrateAddsResultColumn(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "leans.db")

	// 构造旧版 schema（无 result 列）。
	old := `CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
	CREATE TABLE history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		subject TEXT NOT NULL,
		question TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
	);`
	legacy := openRaw(t, dbPath)
	if _, err := legacy.Exec(old); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	legacy.Close()

	// 用 NewStore 打开，触发 migration。
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore on legacy db: %v", err)
	}
	defer store.Close()

	if _, err := store.AddHistory(model.HistoryItem{Subject: "s", Question: "q", Category: "c", Result: `{"ok":1}`}); err != nil {
		t.Fatalf("AddHistory after migrate: %v", err)
	}
	got, err := store.GetHistory(1)
	if err != nil {
		t.Fatalf("GetHistory after migrate: %v", err)
	}
	if got.Result != `{"ok":1}` {
		t.Errorf("result not persisted: %q", got.Result)
	}
}
