package storage

import (
	"encoding/json"
	"fmt"

	"leans/model"
)

// settingsKey 保存多供应商设置的完整 JSON；旧版本按字段拆成多行存储，
// 读取时自动迁移为单供应商，保存时删除旧行。
const settingsKey = "ai_settings"

func (s *Store) GetSettings() (model.AISettings, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return model.AISettings{}, err
	}
	defer rows.Close()

	kv := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return model.AISettings{}, err
		}
		kv[k] = v
	}
	if err := rows.Err(); err != nil {
		return model.AISettings{}, err
	}

	if raw, ok := kv[settingsKey]; ok {
		var out model.AISettings
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return model.AISettings{}, fmt.Errorf("parse settings: %w", err)
		}
		return out, nil
	}
	return migrateLegacySettings(kv), nil
}

// migrateLegacySettings 把旧版单供应商字段迁移为多供应商结构。
func migrateLegacySettings(kv map[string]string) model.AISettings {
	p := model.AIProvider{
		ID:      "default",
		Name:    kv["provider"],
		BaseURL: kv["base_url"],
		APIKey:  kv["api_key"],
	}
	if p.Name == "" {
		p.Name = "默认供应商"
	}
	if m := kv["model"]; m != "" {
		p.Models = []string{m}
	}
	if p.BaseURL == "" && p.APIKey == "" && len(p.Models) == 0 {
		return model.AISettings{}
	}
	return model.AISettings{
		ActiveProviderID: p.ID,
		ActiveModel:      kv["model"],
		Providers:        []model.AIProvider{p},
	}
}

// SaveSettings 以整体 JSON 持久化多供应商设置，并清理旧版字段行。
func (s *Store) SaveSettings(set model.AISettings) error {
	data, err := json.Marshal(set)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		settingsKey, string(data),
	); err != nil {
		return err
	}
	for _, k := range []string{"provider", "api_key", "base_url", "model"} {
		if _, err := tx.Exec(`DELETE FROM settings WHERE key = ?`, k); err != nil {
			return err
		}
	}
	return tx.Commit()
}
