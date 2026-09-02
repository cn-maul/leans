package model

import "encoding/json"

type AnalyzeRequest struct {
	Subject string `json:"subject" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// Highlight 是题目文段中的一处标注。Module 决定它在左右栏的颜色归属
// （category=题型判断、rule=适用规则、annotation=答案/思路、error=转折/错误、info=关键信息），
// Color 为兼容旧数据保留。
type Highlight struct {
	Text        string `json:"text"`
	Type        string `json:"type"`
	Color       string `json:"color,omitempty"`
	Module      string `json:"module,omitempty"`
	Location    string `json:"location"`
	Explanation string `json:"explanation"`
}

// Rule 是一条适用规则，Marks 是该规则在题目文段中命中的词句标注。
type Rule struct {
	Name    string      `json:"name"`
	Section string      `json:"section"`
	Usage   string      `json:"usage"`
	Example string      `json:"example"`
	Marks   []Highlight `json:"marks,omitempty"`
}

// AnalysisMeta 记录一次分析的耗时与 token 用量。
type AnalysisMeta struct {
	ElapsedMS        int64 `json:"elapsed_ms"`
	PromptTokens     int   `json:"prompt_tokens"`
	CompletionTokens int   `json:"completion_tokens"`
	TotalTokens      int   `json:"total_tokens"`
}

type AnalyzeResponse struct {
	Category    string       `json:"category"`
	SubCategory string       `json:"sub_category"`
	Answer      string       `json:"answer,omitempty"` // 本题答案
	Basis       []Highlight  `json:"basis,omitempty"`  // 题型判断依据（词句）
	Rules       []Rule       `json:"rules,omitempty"`  // 适用规则（含命中词句）
	Annotation  string       `json:"annotation,omitempty"`
	Highlights  []Highlight  `json:"highlights,omitempty"`
	Meta        AnalysisMeta `json:"meta"`

	// 兼容旧版字段
	Techniques []TechniqueInfo  `json:"techniques,omitempty"`
	Applicable []ApplicableRule `json:"applicable,omitempty"`
}

type TechniqueInfo struct {
	Name        string `json:"name"`
	Section     string `json:"section"`
	Description string `json:"description"`
}

type ApplicableRule struct {
	RuleName string `json:"rule_name"`
	Section  string `json:"section"`
	Usage    string `json:"usage"`
	Example  string `json:"example"`
}

type Subject struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FilePath string `json:"-"`
	Content  string `json:"-"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

// ToJSON serializes the response for persistence in history records.
func (r *AnalyzeResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
