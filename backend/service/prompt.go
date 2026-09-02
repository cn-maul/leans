package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"leans/model"
	"leans/subject"
)

// analysisPrompt 是"正文"模板：只描述结果结构与模块/颜色规则。
// 讲义正文与题目由 BuildAnalysisPrompt 以检索后的章节注入。
// 模板刻意精简以控制 token 占用；讲义注入量由 lectureBudget 限制。
const analysisPrompt = `你是公务员考试题目分析专家。根据讲义章节分析题目，只输出 JSON，不要任何多余文字。

## 讲义章节（已按相关度挑选）
%s

## 讲义整体概述
%s

## 题目
%s

## 输出 JSON 结构
{
  "category": "题型大类",
  "sub_category": "具体题型",
  "answer": "答案选项，如 B",
  "basis": [{"text": "判断题型的词句", "location": "题干或选项A/B/C/D", "explanation": "判断理由（≤15字）"}],
  "rules": [{"name": "规则名", "section": "讲义章节", "usage": "怎么套用（≤30字）", "example": "示例（≤15字）", "marks": [{"text": "命中词句", "location": "题干或选项A/B/C/D", "explanation": "为何命中（≤15字）"}]}],
  "annotation": "解题思路（≤150字）：判断题型→套规则→逐项排除→答案依据",
  "highlights": [{"text": "关键词句", "type": "类型（≤6字）", "module": "category|rule|annotation|error|info", "location": "题干或选项A/B/C/D", "explanation": "为何重要（≤15字）"}]
}

## module 颜色
category=蓝(题型依据) rule=绿(规则命中) annotation=紫(答案/主旨) error=红(转折/错误) info=黄(关键信息)

## 原则
1. basis 与各 rule.marks 的词句必须同时出现在 highlights 中且 module 对应（左右颜色一致）
2. text 必须能在题目原文逐字找到；location 只能是"题干"或"选项A"…"选项D"
3. rules 从讲义精准匹配并注明章节
4. 严格控制长度：highlights≤5条、basis≤2条、rules≤2条、每条marks≤2个、annotation≤150字、每个explanation≤15字。宁可精简，不要冗长。`

// BuildAnalysisPrompt 用检索到的相关章节构造 system+user 消息。
// lectureBudget 控制讲义注入的字符上限（token 大头）。
// 概述章节单独进"整体概述"槽位，避免与讲义正文重复注入。
func BuildAnalysisPrompt(subName string, hits []subject.Hit, question string, strictJSON bool, lectureBudget int) []model.ChatMessage {
	var lecture strings.Builder
	budget := lectureBudget
	if budget <= 0 {
		budget = 6000
	}

	var overviewHits []subject.Hit
	for _, h := range hits {
		if isOverviewTitle(h.Title) {
			overviewHits = append(overviewHits, h)
			continue
		}
		block := "### " + h.Title + "\n" + h.Content + "\n\n"
		if len([]rune(block)) > budget {
			block = "### " + h.Title + "\n……（章节过长，已跳过）\n\n"
		}
		budget -= len([]rune(block))
		if budget < 0 {
			break
		}
		lecture.WriteString(block)
	}
	if lecture.Len() == 0 {
		lecture.WriteString("（未检索到与本题直接相关的章节）\n")
	}

	overview := buildOverview(overviewHits)
	body := fmt.Sprintf(analysisPrompt, lecture.String(), overview, question)
	if strictJSON {
		body += "\n【重要】只输出 JSON，禁止 markdown 代码块、注释或多余文字。"
	}

	return []model.ChatMessage{
		{
			Role:    "system",
			Content: fmt.Sprintf("你是一个专业的公务员考试分析助手，擅长%s题目。只输出 JSON。", subName),
		},
		{
			Role:    "user",
			Content: body,
		},
	}
}

// isOverviewTitle 判断章节标题是否属于"总体介绍"性质。
func isOverviewTitle(title string) bool {
	for _, kw := range []string{"概述", "总论", "整体", "总体", "大纲", "导言", "引言"} {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

// buildOverview 汇总概述章节文本，作为全局上下文。
func buildOverview(hits []subject.Hit) string {
	var parts []string
	for _, h := range hits {
		if h.Content != "" {
			parts = append(parts, h.Title+":\n"+h.Content)
		}
	}
	if len(parts) == 0 {
		return "（讲义未提供独立整体概述）"
	}
	return strings.Join(parts, "\n\n")
}

// ParseAnalysisResponse 解析 AI 返回文本为结构化结果，带容错与字段规范化。
func ParseAnalysisResponse(response string) (*model.AnalyzeResponse, error) {
	cleaned := extractJSON(response)
	var result model.AnalyzeResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w", err)
	}

	result.Category = strings.TrimSpace(result.Category)
	result.SubCategory = strings.TrimSpace(result.SubCategory)
	result.Answer = strings.TrimSpace(result.Answer)
	result.Annotation = strings.TrimSpace(result.Annotation)

	for i := range result.Highlights {
		result.Highlights[i].Color = normalizeColor(result.Highlights[i].Color)
		result.Highlights[i].Module = normalizeModule(result.Highlights[i].Module)
		result.Highlights[i].Location = normalizeLocation(result.Highlights[i].Location)
	}
	for i := range result.Basis {
		result.Basis[i].Module = normalizeModule(result.Basis[i].Module)
		result.Basis[i].Location = normalizeLocation(result.Basis[i].Location)
	}
	for i := range result.Rules {
		for j := range result.Rules[i].Marks {
			result.Rules[i].Marks[j].Module = normalizeModule(result.Rules[i].Marks[j].Module)
			result.Rules[i].Marks[j].Location = normalizeLocation(result.Rules[i].Marks[j].Location)
		}
	}

	// 旧版数据兜底：没有新字段时映射旧字段。
	fallbackLegacy(&result)
	return &result, nil
}

// fallbackLegacy 将旧版字段（techniques/applicable）映射到新结构，
// 保证历史记录也能以新三块布局展示。
func fallbackLegacy(r *model.AnalyzeResponse) {
	if len(r.Basis) == 0 && len(r.Highlights) > 0 {
		for _, h := range r.Highlights {
			if h.Module == "category" {
				r.Basis = append(r.Basis, h)
			}
		}
	}
	if len(r.Rules) == 0 && len(r.Applicable) > 0 {
		for _, a := range r.Applicable {
			r.Rules = append(r.Rules, model.Rule{
				Name:    a.RuleName,
				Section: a.Section,
				Usage:   a.Usage,
				Example: a.Example,
			})
		}
	}
}

// extractJSON 从 AI 输出中提取第一个完整 JSON 对象：剥离 markdown 代码块、
// 前导/尾随文字、注释行。用深度计数匹配花括号，支持任意层级嵌套。
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// 去掉 markdown 代码围栏（```json ... ``` 或 ``` ... ```）。
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	// 去掉 // 与 # 注释行（部分模型会在 JSON 里加注释）。
	lines := strings.Split(s, "\n")
	var kept []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") {
			continue
		}
		kept = append(kept, l)
	}
	s = strings.Join(kept, "\n")

	// 深度计数：从第一个 '{' 开始，遇到 '{' 加一、'}' 减一，直到归零。
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				// 模型偶发会在 JSON 对象内部插入裸中文说明（无引号），
				// 逐行净化后再返回。
				return sanitizeJSONBlock(s[start : i+1])
			}
		}
	}
	return sanitizeJSONBlock(s)
}

// sanitizeJSONBlock 移除 JSON 块中字符串之外的裸文本（模型偶发插入的
// 中文说明、多余标点等），同时不触碰字符串内的内容。做法：逐行扫描，
// 跟踪字符串状态，在字符串外只保留 JSON 结构字符与数字，其余截断。
func sanitizeJSONBlock(block string) string {
	lines := strings.Split(block, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		kept = append(kept, trimJSONLine(line))
	}
	return strings.Join(kept, "\n")
}

// trimJSONLine 去掉一行里字符串外多余的裸文本。
func trimJSONLine(line string) string {
	inStr := false
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inStr {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inStr = false
			}
			continue
		}
		// 字符串外：保留结构字符、空白、数字、负号，以及
		// JSON 字面量 true/false/null 的字母。
		if ch == '"' || ch == '{' || ch == '}' || ch == '[' || ch == ']' ||
			ch == ':' || ch == ',' || ch == ' ' || ch == '\t' || ch == '\r' ||
			ch == '-' || (ch >= '0' && ch <= '9') ||
			ch == 't' || ch == 'r' || ch == 'u' || ch == 'e' ||
			ch == 'f' || ch == 'a' || ch == 'l' || ch == 's' || ch == 'n' {
			if ch == '"' {
				inStr = true
			}
			continue
		}
		// 遇到裸文本（中文等）→ 截断该行剩余部分。
		return line[:i]
	}
	return line
}

// normalizeColor 把 AI 可能返回的各种颜色表达归一为标准 key。
func normalizeColor(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	trimmed := strings.TrimPrefix(c, "#")
	switch trimmed {
	case "green", "g", "绿", "绿色", "22c55e", "16a34a":
		return "green"
	case "red", "r", "红", "红色", "ef4444", "dc2626":
		return "red"
	case "blue", "b", "蓝", "蓝色", "3b82f6", "2563eb":
		return "blue"
	case "yellow", "y", "黄", "黄色", "eab308", "ca8a04":
		return "yellow"
	case "purple", "p", "紫", "紫色", "a855f7", "9333ea":
		return "purple"
	}
	return ""
}

// normalizeModule 把 AI 返回的 module 表达归一为标准值。
func normalizeModule(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "category", "题型", "分类", "判断":
		return "category"
	case "rule", "规则", "技巧", "适用":
		return "rule"
	case "annotation", "answer", "答案", "注释", "思路", "主旨":
		return "annotation"
	case "error", "错误", "转折", "否定":
		return "error"
	case "info", "关键", "信息", "条件":
		return "info"
	}
	return ""
}

// normalizeLocation 把"选项A / 选项 a / A / 题干 / 原文"归一为"题干|选项A|..."。
func normalizeLocation(loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return "题干"
	}
	lower := strings.ToLower(loc)
	if strings.Contains(lower, "题干") || strings.Contains(lower, "原文") ||
		strings.Contains(lower, "stem") || strings.Contains(lower, "题目") {
		return "题干"
	}
	// 匹配 "选项A"、"A"、"选项 A" 等。
	for _, ch := range []byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'} {
		if strings.Contains(strings.ToUpper(loc), string(ch)) {
			return "选项" + string(ch)
		}
	}
	return "题干"
}
