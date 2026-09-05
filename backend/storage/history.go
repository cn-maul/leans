package storage

import "leans/model"

// AddHistory inserts a new analysis history record from the populated fields
// of the item (ID/CreatedAt ignored). result is the raw JSON of the full
// analysis; model/tokens/耗时字段供历史列表和统计页使用。
func (s *Store) AddHistory(rec model.HistoryItem) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO history (subject, question, category, result, model, tokens, elapsed_ms, first_token_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.Subject, rec.Question, rec.Category, rec.Result, rec.Model,
		rec.Tokens, rec.ElapsedMS, rec.FirstTokenMS,
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
		`SELECT id, subject, question, category, model, tokens, elapsed_ms, first_token_ms, created_at
		 FROM history ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.HistoryItem
	for rows.Next() {
		var it model.HistoryItem
		if err := rows.Scan(&it.ID, &it.Subject, &it.Question, &it.Category, &it.Model,
			&it.Tokens, &it.ElapsedMS, &it.FirstTokenMS, &it.CreatedAt); err != nil {
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
		`SELECT id, subject, question, category, result, model, tokens, elapsed_ms, first_token_ms, created_at
		 FROM history WHERE id = ?`,
		id,
	).Scan(&it.ID, &it.Subject, &it.Question, &it.Category, &it.Result, &it.Model,
		&it.Tokens, &it.ElapsedMS, &it.FirstTokenMS, &it.CreatedAt)
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

// GetStats aggregates history records into the summary shown on the stats page.
// 平均首字只统计有首字耗时记录（流式请求）的行。
func (s *Store) GetStats() (*model.Stats, error) {
	var st model.Stats
	err := s.db.QueryRow(
		`SELECT COUNT(*),
			COALESCE(SUM(tokens), 0),
			COALESCE(AVG(tokens), 0),
			COALESCE(AVG(CASE WHEN first_token_ms > 0 THEN first_token_ms END), 0),
			COALESCE(SUM(elapsed_ms), 0)
		 FROM history`,
	).Scan(&st.TotalQuestions, &st.TotalTokens, &st.AvgTokens, &st.AvgFirstTokenMS, &st.TotalElapsedMS)
	if err != nil {
		return nil, err
	}
	return &st, nil
}
