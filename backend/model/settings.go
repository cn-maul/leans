package model

// 支持的接口协议。auto 表示按端点主动探测（rosetta DetectProtocol），
// 其余值与 rosetta 的 Protocol 常量一一对应。
const (
	ProtocolAuto            = "auto"
	ProtocolOpenAIChat      = "openai-chat"      // OpenAI Chat Completions（绝大多数兼容服务）
	ProtocolOpenAIResponses = "openai-responses" // OpenAI Responses API
	ProtocolAnthropic       = "anthropic"        // Anthropic Messages API
)

// ValidProtocol 报告协议取值是否合法（空串视为 auto）。
func ValidProtocol(p string) bool {
	switch p {
	case "", ProtocolAuto, ProtocolOpenAIChat, ProtocolOpenAIResponses, ProtocolAnthropic:
		return true
	}
	return false
}

// AIProvider 是一个可切换的 AI 服务接入点：一个 endpoint + 凭证 +
// 该服务下可用的模型列表。模型可手动维护，也可从服务端在线获取。
type AIProvider struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	BaseURL  string   `json:"base_url"`
	APIKey   string   `json:"api_key"`
	Protocol string   `json:"protocol"`
	Models   []string `json:"models"`
}

// AISettings 多供应商设置。ActiveProviderID/ActiveModel 是主界面二级
// 下拉的当前选择；Providers 为空表示尚未配置任何 AI 服务。
type AISettings struct {
	ActiveProviderID string       `json:"active_provider_id"`
	ActiveModel      string       `json:"active_model"`
	Providers        []AIProvider `json:"providers"`
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
