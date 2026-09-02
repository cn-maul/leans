package storage

import "leans/model"

// GetSettings reads persisted AI settings from the DB. Missing keys are
// returned as empty strings so the caller can fall back to defaults.
func (s *Store) GetSettings() (model.AISettings, error) {
	var out model.AISettings
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return out, err
	}
	defer rows.Close()

	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "provider":
			out.Provider = v
		case "api_key":
			out.APIKey = v
		case "base_url":
			out.BaseURL = v
		case "model":
			out.Model = v
		}
	}
	return out, rows.Err()
}

// SaveSettings upserts all AI setting fields.
func (s *Store) SaveSettings(set model.AISettings) error {
	vals := map[string]string{
		"provider": set.Provider,
		"api_key":  set.APIKey,
		"base_url": set.BaseURL,
		"model":    set.Model,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for k, v := range vals {
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			k, v,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
