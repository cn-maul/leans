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

	id, err := s.AddHistory("言语理解", "题目一", "中心理解", `{"category":"中心理解"}`)
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
		if _, err := s.AddHistory("s", "q", "c", "{}"); err != nil {
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

	if _, err := store.AddHistory("s", "q", "c", `{"ok":1}`); err != nil {
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
