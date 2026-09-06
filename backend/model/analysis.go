package model

import "encoding/json"

type AnalyzeRequest struct {
	Subject string `json:"subject" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// Question 是结构化题目对象。输入层仍保持粘贴文本，后端负责把文本解析成
// 这个结构，供题库、学习记录和后续检索复用。
type Question struct {
	ID          int64    `json:"id,omitempty"`
	Subject     string   `json:"subject"`
	Stem        string   `json:"stem"`
	Options     []string `json:"options,omitempty"`
	Answer      string   `json:"answer,omitempty"`
	Category    string   `json:"category,omitempty"`
	SubCategory string   `json:"sub_category,omitempty"`
	Source      string   `json:"source,omitempty"`
	Year        string   `json:"year,omitempty"`
	Difficulty  string   `json:"difficulty,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

// KnowledgeUnit 是从讲义中抽取的知识单元，比整章切片更适合检索和 Prompt 注入。
type KnowledgeUnit struct {
	ID            string   `json:"id,omitempty"`
	Subject       string   `json:"subject"`
	Title         string   `json:"title"`
	Section       string   `json:"section"`
	Content       string   `json:"content"`
	Kind          string   `json:"kind"`
	Keywords      []string `json:"keywords,omitempty"`
	Rule          string   `json:"rule,omitempty"`
	Example       string   `json:"example,omitempty"`
	Trap          string   `json:"trap,omitempty"`
	SourceSection string   `json:"source_section"`
}

// StudyRecord 表示一次学习行为：答对、答错、复习等。本地单机版保存这些记录，
// 用于薄弱点统计和后续推荐。
type StudyRecord struct {
	ID           int64  `json:"id,omitempty"`
	Subject      string `json:"subject"`
	QuestionID   int64  `json:"question_id"`
	HistoryID    int64  `json:"history_id"`
	UserAnswer   string `json:"user_answer"`
	Correct      bool   `json:"correct"`
	Difficulty   string `json:"difficulty,omitempty"`
	ReviewAt     string `json:"review_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	SkillChanged string `json:"skill_change,omitempty"`
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

// Evidence 表示一个结论的证据来源，用来防止 AI 结果像黑盒。
type Evidence struct {
	Source     string  `json:"source"`
	Section    string  `json:"section"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Rule 是一条适用规则。Application 是该技巧在本题中的具体用法（新版字段），
// Usage 为兼容旧数据保留；解析时两者会同步。
type Rule struct {
	Name        string      `json:"name"`
	Section     string      `json:"section"`
	Usage       string      `json:"usage"`
	Application string      `json:"application,omitempty"`
	Example     string      `json:"example,omitempty"`
	Marks       []Highlight `json:"marks,omitempty"`
	Evidence    []Evidence  `json:"evidence,omitempty"`
	Confidence  float64     `json:"confidence,omitempty"`
}

// TypeJudgment 是"题型判断"固定区块的结构化输出。
type TypeJudgment struct {
	Category    string      `json:"category"`
	SubCategory string      `json:"sub_category"`
	Basis       []Highlight `json:"basis,omitempty"`
}

// TechniqueJudgment 是"技巧判断"固定区块的结构化输出。
type TechniqueJudgment struct {
	Rules []Rule `json:"rules,omitempty"`
}

// ReasoningStep 拆出解题推理步骤，供前端把"怎么想"显式展示。
type ReasoningStep struct {
	Step     string     `json:"step"`
	Detail   string     `json:"detail,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// OptionAnalysis 解释每个选项为什么对或错。
type OptionAnalysis struct {
	Option    string `json:"option"`
	Text      string `json:"text"`
	Judgement string `json:"judgement"`
	Reason    string `json:"reason,omitempty"`
}

// AnalysisMeta 记录一次分析的耗时与 token 用量。
// FirstTokenMS 为首个内容 token 的响应耗时（流式请求才有）；Model 为实际使用的模型名。
type AnalysisMeta struct {
	ElapsedMS        int64  `json:"elapsed_ms"`
	FirstTokenMS     int64  `json:"first_token_ms,omitempty"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	Model            string `json:"model,omitempty"`
}

// AnalysisQuality 提供 AI 结果的质量信号，避免把所有结果都当作确定结论。
type AnalysisQuality struct {
	Confidence  float64 `json:"confidence,omitempty"`
	NeedsReview bool    `json:"needs_review,omitempty"`
	Notes       string  `json:"notes,omitempty"`
}

type AnalyzeResponse struct {
	// TypeJudgment / TechniqueJudgment 是固定区域使用的首选字段；
	// 顶层 Category/Basis/Rules 等字段为兼容旧版保留，解析时自动同步。
	TypeJudgment      *TypeJudgment      `json:"type_judgment,omitempty"`
	TechniqueJudgment *TechniqueJudgment `json:"technique_judgment,omitempty"`
	Category          string             `json:"category"`
	SubCategory       string             `json:"sub_category"`
	Answer            string             `json:"answer,omitempty"` // 本题答案
	Basis             []Highlight        `json:"basis,omitempty"`  // 题型判断依据（词句）
	Rules             []Rule             `json:"rules,omitempty"`  // 适用规则（含命中词句）
	Annotation        string             `json:"annotation,omitempty"`
	Highlights        []Highlight        `json:"highlights,omitempty"`
	Question          *Question          `json:"question,omitempty"`
	ReasoningSteps    []ReasoningStep    `json:"reasoning_steps,omitempty"`
	OptionAnalysis    []OptionAnalysis   `json:"option_analysis,omitempty"`
	Evidence          []Evidence         `json:"evidence,omitempty"`
	Quality           *AnalysisQuality   `json:"quality,omitempty"`
	Meta              AnalysisMeta       `json:"meta"`

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

// ChatMessage 是应用层的对话消息；线上协议的编解码由 rosetta 负责。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage 记录一次调用的 token 用量。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToJSON serializes the response for persistence in history records.
func (r *AnalyzeResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
