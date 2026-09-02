package storage

import "leans/model"

// AddHistory inserts a new analysis history record. result is the raw JSON
// of the full analysis, persisted so the frontend can restore it later.
func (s *Store) AddHistory(subject, question, category, result string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO history (subject, question, category, result) VALUES (?, ?, ?, ?)",
		subject, question, category, result,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListHistory returns the most recent history records (summary fields only,
// without the full result JSON to keep the list light).
func (s *Store) ListHistory(limit int) ([]model.HistoryItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(
		"SELECT id, subject, question, category, created_at FROM history ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.HistoryItem
	for rows.Next() {
		var it model.HistoryItem
		if err := rows.Scan(&it.ID, &it.Subject, &it.Question, &it.Category, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetHistory returns a single record including the stored analysis result.
func (s *Store) GetHistory(id int64) (*model.HistoryItem, error) {
	var it model.HistoryItem
	err := s.db.QueryRow(
		"SELECT id, subject, question, category, result, created_at FROM history WHERE id = ?",
		id,
	).Scan(&it.ID, &it.Subject, &it.Question, &it.Category, &it.Result, &it.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// ClearHistory removes all history records.
func (s *Store) ClearHistory() error {
	_, err := s.db.Exec("DELETE FROM history")
	return err
}
