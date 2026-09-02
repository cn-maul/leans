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
	ID        int64  `json:"id"`
	Subject   string `json:"subject"`
	Question  string `json:"question"`
	Category  string `json:"category"`
	Result    string `json:"result,omitempty"`
	CreatedAt string `json:"created_at"`
}