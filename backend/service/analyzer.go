package service

import (
	"context"
	"errors"
	"fmt"
	"leans/config"
	"leans/model"
	"leans/storage"
	"leans/subject"
	"log"
	"time"
)

// ErrNotConfigured 表示 AI 设置缺失或无效，是用户可纠正的状态
// （handler 据此映射为 4xx，而不是 500）。
var ErrNotConfigured = errors.New("AI 未配置")

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

// StreamCallbacks 接收流式分析过程中的事件。
type StreamCallbacks struct {
	// OnModel 在确定实际使用的模型后调用一次。
	OnModel func(modelName string)
	// OnDelta 在每段内容增量到达时调用（原始 JSON 文本片段）。
	OnDelta func(text string)
	// OnStatus 通知前端流程状态变化（如解析失败重试）。
	OnStatus func(message string)
}

// buildPrompt 检索讲义并构造分析消息，供流式/非流式共用。
func (a *Analyzer) buildPrompt(opt AnalyzeOption) ([]model.ChatMessage, error) {
	sub, err := a.subjects.Get(opt.Subject)
	if err != nil {
		return nil, err
	}
	hits := a.subjects.Search(opt.Subject, opt.Content, subject.Retrieval{
		MaxSections: a.cfg.MaxSections,
		MaxChars:    a.cfg.MaxChars,
		OverviewMax: a.cfg.OverviewMax,
	})
	typeCatalog := a.subjects.TypeCatalog(opt.Subject, 30)
	return BuildAnalysisPrompt(sub.Name, hits, typeCatalog, opt.Content, false, a.cfg.LectureBudget), nil
}

// fillMeta 统一填充 meta 并入库。
func (a *Analyzer) fillMeta(opt AnalyzeOption, result *model.AnalyzeResponse, start time.Time, firstTokenMS int64, usage *model.Usage, modelName string) {
	result.Meta = model.AnalysisMeta{
		ElapsedMS:        time.Since(start).Milliseconds(),
		FirstTokenMS:     firstTokenMS,
		PromptTokens:     usageTokens(usage).prompt,
		CompletionTokens: usageTokens(usage).completion,
		TotalTokens:      usageTokens(usage).total,
		Model:            modelName,
	}
	a.saveHistory(opt, result)
}

// Analyze runs the full pipeline. The returned result is already normalized.
// ctx 取消时中止 AI 调用（返回 context.Canceled 包装错误）。
func (a *Analyzer) Analyze(ctx context.Context, opt AnalyzeOption) (*model.AnalyzeResponse, error) {
	if a.ai == nil {
		return nil, fmt.Errorf("AI 服务未配置")
	}
	if opt.Subject == "" || opt.Content == "" {
		return nil, fmt.Errorf("题目或科目不能为空")
	}

	resolved, err := a.ai.Resolve()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	messages, err := a.buildPrompt(opt)
	if err != nil {
		return nil, err
	}

	var lastErr error
	var usage *model.Usage
	for attempt := 0; attempt <= 1; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt > 0 {
			// 重试时在消息末尾追加严格 JSON 提醒。
			messages = appendStrictJSONReminder(messages)
		}
		response, u, err := a.ai.ChatCompletion(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("AI 调用失败: %w", err)
		}
		if u != nil {
			usage = u
		}
		result, err := ParseAnalysisResponse(response)
		if err == nil {
			a.fillMeta(opt, result, start, 0, usage, resolved.Model)
			return result, nil
		}
		lastErr = fmt.Errorf("解析 AI 输出失败: %w", err)
	}
	return nil, lastErr
}

// AnalyzeStream 流式分析：内容增量通过 cb.OnDelta 实时回调；最终返回与
// Analyze 相同的规范化结果并入库。ctx 取消（如用户中止）时流随之终止，
// 不会入库。流式输出解析失败时自动降级为一次非流式重试（delta 已发出，
// 前端以最终 done 结果为准）。
func (a *Analyzer) AnalyzeStream(ctx context.Context, opt AnalyzeOption, cb StreamCallbacks) (*model.AnalyzeResponse, error) {
	if a.ai == nil {
		return nil, fmt.Errorf("AI 服务未配置")
	}
	if opt.Subject == "" || opt.Content == "" {
		return nil, fmt.Errorf("题目或科目不能为空")
	}

	resolved, err := a.ai.Resolve()
	if err != nil {
		return nil, err
	}
	if cb.OnModel != nil {
		cb.OnModel(resolved.Model)
	}

	start := time.Now()
	messages, err := a.buildPrompt(opt)
	if err != nil {
		return nil, err
	}

	content, usage, firstTokenMS, serr := a.ai.ChatCompletionStream(ctx, messages, cb.OnDelta)
	if serr != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", serr)
	}

	result, err := ParseAnalysisResponse(content)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if cb.OnStatus != nil {
			cb.OnStatus("输出解析失败，正在重试…")
		}
		retryMsgs := appendStrictJSONReminder(messages)
		var response string
		var u *model.Usage
		response, u, err = a.ai.ChatCompletion(ctx, retryMsgs)
		if err != nil {
			return nil, fmt.Errorf("AI 调用失败: %w", err)
		}
		if u != nil {
			usage = u
		}
		result, err = ParseAnalysisResponse(response)
		if err != nil {
			return nil, fmt.Errorf("解析 AI 输出失败: %w", err)
		}
	}

	a.fillMeta(opt, result, start, firstTokenMS, usage, resolved.Model)
	return result, nil
}

// appendStrictJSONReminder 在消息末尾追加严格 JSON 输出提醒（重试时使用）。
func appendStrictJSONReminder(messages []model.ChatMessage) []model.ChatMessage {
	out := make([]model.ChatMessage, len(messages))
	copy(out, messages)
	out[len(out)-1] = model.ChatMessage{
		Role: out[len(out)-1].Role,
		Content: out[len(out)-1].Content +
			"\n【重要】只输出 JSON，禁止 markdown 代码块、注释或多余文字。",
	}
	return out
}

// saveHistory persists a history record with the full result JSON.
// 失败仅记日志：历史写入失败不应影响分析结果返回。
func (a *Analyzer) saveHistory(opt AnalyzeOption, result *model.AnalyzeResponse) {
	if a.store == nil || !opt.SaveHistory {
		return
	}
	jsonStr, err := result.ToJSON()
	if err != nil {
		log.Printf("save history: marshal result: %v", err)
		return
	}
	if _, err := a.store.AddHistory(model.HistoryItem{
		Subject:      opt.Subject,
		Question:     opt.Content,
		Category:     result.Category,
		Result:       jsonStr,
		Model:        result.Meta.Model,
		Tokens:       result.Meta.TotalTokens,
		ElapsedMS:    result.Meta.ElapsedMS,
		FirstTokenMS: result.Meta.FirstTokenMS,
	}); err != nil {
		log.Printf("save history: %v", err)
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
