package service

import (
	"fmt"
	"leans/config"
	"leans/model"
	"leans/storage"
	"leans/subject"
	"time"
)

// Analyzer orchestrates lecture retrieval + AI completion + response parsing.
type Analyzer struct {
	ai       *AIService
	subjects *subject.Store
	store    *storage.Store
	cfg      config.Analysis
}

func NewAnalyzer(ai *AIService, subjects *subject.Store, store *storage.Store, cfg config.Analysis) *Analyzer {
	return &Analyzer{
		ai:       ai,
		subjects: subjects,
		store:    store,
		cfg:      cfg,
	}
}

// AnalyzeOption controls one analysis run.
type AnalyzeOption struct {
	Subject     string
	Content     string
	SaveHistory bool
}

// Analyze runs the full pipeline. The returned result is already normalized.
func (a *Analyzer) Analyze(opt AnalyzeOption) (*model.AnalyzeResponse, error) {
	if a.ai == nil {
		return nil, fmt.Errorf("AI 服务未配置")
	}
	if err := a.ai.Validate(); err != nil {
		return nil, err
	}
	if opt.Subject == "" || opt.Content == "" {
		return nil, fmt.Errorf("题目或科目不能为空")
	}

	maxSections := a.cfg.MaxSections
	maxChars := a.cfg.MaxChars
	overviewMax := a.cfg.OverviewMax
	lectureBudget := a.cfg.LectureBudget

	start := time.Now()
	sub, err := a.subjects.Get(opt.Subject)
	if err != nil {
		return nil, err
	}

	hits := a.subjects.Search(opt.Subject, opt.Content, subject.Retrieval{
		MaxSections: maxSections,
		MaxChars:    maxChars,
		OverviewMax: overviewMax,
	})

	var lastErr error
	var usage *model.Usage
	for attempt := 0; attempt <= 1; attempt++ {
		messages := BuildAnalysisPrompt(sub.Name, hits, opt.Content, attempt > 0, lectureBudget)
		response, u, err := a.ai.ChatCompletion(messages)
		if err != nil {
			return nil, fmt.Errorf("AI 调用失败: %w", err)
		}
		if u != nil {
			usage = u
		}
		result, err := ParseAnalysisResponse(response)
		if err == nil {
			result.Meta = model.AnalysisMeta{
				ElapsedMS:        time.Since(start).Milliseconds(),
				PromptTokens:     usageTokens(usage).prompt,
				CompletionTokens: usageTokens(usage).completion,
				TotalTokens:      usageTokens(usage).total,
			}
			a.saveHistory(opt, result)
			return result, nil
		}
		lastErr = fmt.Errorf("解析 AI 输出失败: %w", err)
	}
	return nil, lastErr
}

// saveHistory persists a history record with the full result JSON.
func (a *Analyzer) saveHistory(opt AnalyzeOption, result *model.AnalyzeResponse) {
	if a.store == nil || !opt.SaveHistory {
		return
	}
	if jsonStr, err := result.ToJSON(); err == nil {
		_, _ = a.store.AddHistory(opt.Subject, opt.Content, result.Category, jsonStr)
	}
}

type tokenUsage struct{ prompt, completion, total int }

// usageTokens 从可空的 API usage 中安全取值。
func usageTokens(u *model.Usage) tokenUsage {
	if u == nil {
		return tokenUsage{}
	}
	return tokenUsage{prompt: u.PromptTokens, completion: u.CompletionTokens, total: u.TotalTokens}
}
