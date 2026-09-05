package model

// AISettings represents the AI provider configuration editable from the UI.
type AISettings struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

// History record of a single analysis run.
type HistoryItem struct {
	ID           int64  `json:"id"`
	Subject      string `json:"subject"`
	Question     string `json:"question"`
	Category     string `json:"category"`
	Result       string `json:"result,omitempty"`
	Model        string `json:"model,omitempty"`
	Tokens       int    `json:"tokens"`
	ElapsedMS    int64  `json:"elapsed_ms"`
	FirstTokenMS int64  `json:"first_token_ms,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// Stats 汇总历史记录的统计信息，供统计页展示。
type Stats struct {
	TotalQuestions  int64   `json:"total_questions"`
	TotalTokens     int64   `json:"total_tokens"`
	AvgTokens       float64 `json:"avg_tokens"`
	AvgFirstTokenMS float64 `json:"avg_first_token_ms"`
	TotalElapsedMS  int64   `json:"total_elapsed_ms"`
}
