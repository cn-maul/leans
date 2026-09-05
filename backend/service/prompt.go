package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"leans/model"
	"leans/subject"
)

// analysisPrompt 是"正文"模板：只描述结果结构与模块/颜色规则。
// 题型清单、讲义正文与题目由 BuildAnalysisPrompt 注入。
// 模板刻意精简以控制 token 占用；讲义注入量由 lectureBudget 限制。
const analysisPrompt = `你是公务员考试题目分析专家。根据讲义章节分析题目，只输出 JSON，不要任何多余文字。

## 无视觉约束
你只能看到题目文本，看不到任何图片、图表、截图、公式或版式。不要假设存在图像内容；若文本包含图片/图表占位符或信息明显缺失，基于可见文字判断，并在 annotation 中如实说明"无法从文本判断"，不要编造。

## 题型清单
%s

## 讲义章节（已按相关度挑选）
%s

## 讲义整体概述
%s

## 题目
%s

## 输出 JSON 结构
{
  "type_judgment": {
    "category": "题型大类（必须从题型清单中选择）",
    "sub_category": "具体题型（尽量从题型清单中选择）"
  },
  "technique_judgment": {
    "rules": [{"name": "规则名", "section": "讲义章节", "usage": "该技巧在这道题里怎么用：结合题干的具体词句说明套用过程（≤60字）"}]
  },
  "answer": "答案选项，如 B",
  "annotation": "解题思路（≤150字）：逐项分析选项对错与答案依据；不要重复 rules.usage 已说明的技巧套用过程",
  "highlights": [{"text": "关键词句", "type": "类型（≤6字）", "module": "category|rule|annotation|error|info", "location": "题干或选项A/B/C/D", "explanation": "为何重要（≤15字）"}]
}

## 原则
1. 先判断题型，再匹配技巧；highlights 覆盖题型判断、技巧命中、答案主旨、错误/转折、关键信息等词句，module 与含义一一对应
2. text 必须能在题目原文逐字找到；location 只能是"题干"或"选项A"…"选项D"
3. rules 只能引用"讲义章节"中实际出现的内容并注明章节；rules 的每个对象只允许 name、section、usage 三个字段；usage 必填，必须结合本题题干词句说明技巧怎么用，禁止照抄讲义原文；讲义没有对应技巧时 rules 返回空数组，禁止编造规则
4. category 必须从题型清单中选择；清单没有明确对应时选择最接近的，并在 highlights 中用 module=category 标注判断词句
5. 严格控制长度：highlights≤5条、rules≤2条、每个usage≤60字、annotation≤150字、每个explanation≤15字。宁可精简，不要冗长。`

// BuildAnalysisPrompt 用检索到的相关章节构造 system+user 消息。
// lectureBudget 控制讲义注入的字符上限（token 大头）。
// 概述章节单独进"整体概述"槽位，避免与讲义正文重复注入。
func BuildAnalysisPrompt(subName string, hits []subject.Hit, typeCatalog []string, question string, strictJSON bool, lectureBudget int) []model.ChatMessage {
	var lecture strings.Builder
	budget := lectureBudget
	if budget <= 0 {
		budget = 6000
	}

	var overviewHits []subject.Hit
	for _, h := range hits {
		if subject.IsOverviewTitle(h.Title) {
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
	typeList := formatTypeCatalog(typeCatalog)
	body := fmt.Sprintf(analysisPrompt, typeList, lecture.String(), overview, question)
	if strictJSON {
		body += "\n【重要】只输出 JSON，禁止 markdown 代码块、注释或多余文字。"
	}

	return []model.ChatMessage{
		{
			Role:    "system",
			Content: fmt.Sprintf("你是一个专业的公务员考试分析助手，擅长%s题目。只输出 JSON。当前任务只有纯文本，没有图片、图表或公式输入。", subName),
		},
		{
			Role:    "user",
			Content: body,
		},
	}
}

// formatTypeCatalog 把题型清单压成一行，控制 prompt 的 token 占用。
func formatTypeCatalog(items []string) string {
	if len(items) == 0 {
		return "（未提供题型清单，请根据讲义章节自行归纳）"
	}
	return strings.Join(items, "、")
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
	// 部分模型输出 JSON 时会漏掉键值对之间的逗号，先做无副作用的补全修复。
	cleaned = repairJSONCommas(cleaned)
	var result model.AnalyzeResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w", err)
	}

	// 旧版数据兜底：没有新字段时映射旧字段；随后统一同步到固定区块。
	fallbackLegacy(&result)
	syncJudgments(&result)
	return &result, nil
}

// syncJudgments 让 type_judgment / technique_judgment 与旧版顶层字段保持一致，
// 并对两边的标注做统一的颜色、模块、位置规范化。
func syncJudgments(r *model.AnalyzeResponse) {
	r.Answer = strings.TrimSpace(r.Answer)
	r.Annotation = strings.TrimSpace(r.Annotation)
	r.Highlights = normalizeHighlights(r.Highlights)

	if r.TypeJudgment != nil {
		r.TypeJudgment.Category = strings.TrimSpace(r.TypeJudgment.Category)
		r.TypeJudgment.SubCategory = strings.TrimSpace(r.TypeJudgment.SubCategory)
		r.TypeJudgment.Basis = normalizeHighlights(r.TypeJudgment.Basis)
		r.Category = r.TypeJudgment.Category
		r.SubCategory = r.TypeJudgment.SubCategory
		r.Basis = r.TypeJudgment.Basis
	} else {
		r.TypeJudgment = &model.TypeJudgment{
			Category:    strings.TrimSpace(r.Category),
			SubCategory: strings.TrimSpace(r.SubCategory),
			Basis:       normalizeHighlights(r.Basis),
		}
	}

	if r.TechniqueJudgment != nil {
		r.TechniqueJudgment.Rules = normalizeRules(r.TechniqueJudgment.Rules)
		r.Rules = r.TechniqueJudgment.Rules
	} else {
		r.TechniqueJudgment = &model.TechniqueJudgment{Rules: normalizeRules(r.Rules)}
	}
}

// normalizeHighlights 对一组标注统一做颜色、模块、位置规范化。
func normalizeHighlights(hs []model.Highlight) []model.Highlight {
	for i := range hs {
		hs[i].Color = normalizeColor(hs[i].Color)
		hs[i].Module = normalizeModule(hs[i].Module)
		hs[i].Location = normalizeLocation(hs[i].Location)
	}
	return hs
}

// normalizeRules 对每条规则的命中标注做统一规范化。
func normalizeRules(rs []model.Rule) []model.Rule {
	for i := range rs {
		rs[i].Marks = normalizeHighlights(rs[i].Marks)
		// 统一用法字段：application 为空时回填 usage，前端只读 application。
		if rs[i].Application == "" {
			rs[i].Application = rs[i].Usage
		}
	}
	return rs
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

// repairJSONCommas 在字符串外相邻的 JSON 值之间补上缺失的逗号。
// 部分模型输出 JSON 时会漏掉 key:value 对之间的逗号。
// 对合法 JSON 无副作用：合法 JSON 中相邻值之间必然已有逗号，不会触发插入。
func repairJSONCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 16)
	inString := false
	escaped := false
	lastSig := byte(0)   // 字符串外最近的有效字符
	sawSpace := false    // lastSig 之后是否出现过空白（"1 2" 缺逗号 vs "12" 合法数字）
	var pendingWS []byte // 暂存的空白，确定是否补逗号后再输出

	needComma := func() bool {
		switch {
		case lastSig == '"' || lastSig == '}' || lastSig == ']':
			return true
		case lastSig >= '0' && lastSig <= '9':
			return true
		case lastSig == 'e' || lastSig == 'l': // true/false/null 的结尾
			return true
		}
		return false
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			b.WriteByte(ch)
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
				lastSig = '"'
				sawSpace = false
			}
			continue
		}
		switch ch {
		case ' ', '\t', '\r', '\n':
			if lastSig != 0 {
				sawSpace = true
				pendingWS = append(pendingWS, ch)
			} else {
				b.WriteByte(ch)
			}
		case '"', '{', '[':
			if needComma() {
				b.WriteByte(',')
			}
			b.Write(pendingWS)
			pendingWS = pendingWS[:0]
			b.WriteByte(ch)
			if ch == '"' {
				inString = true
			} else {
				lastSig = ch
				sawSpace = false
			}
		default:
			// "1 2" 形式的数字缺逗号（要求之间有空白，避免拆散 "12"）。
			if sawSpace && lastSig >= '0' && lastSig <= '9' &&
				(ch == '-' || (ch >= '0' && ch <= '9')) {
				b.WriteByte(',')
			}
			b.Write(pendingWS)
			pendingWS = pendingWS[:0]
			b.WriteByte(ch)
			lastSig = ch
			sawSpace = false
		}
	}
	b.Write(pendingWS)
	return b.String()
}

// colorAliases 把 AI 可能返回的各种颜色表达归一为标准 key。
var colorAliases = map[string]string{
	"green": "green", "g": "green", "绿": "green", "绿色": "green",
	"22c55e": "green", "16a34a": "green",
	"red": "red", "r": "red", "红": "red", "红色": "red",
	"ef4444": "red", "dc2626": "red",
	"blue": "blue", "b": "blue", "蓝": "blue", "蓝色": "blue",
	"3b82f6": "blue", "2563eb": "blue",
	"yellow": "yellow", "y": "yellow", "黄": "yellow", "黄色": "yellow",
	"eab308": "yellow", "ca8a04": "yellow",
	"purple": "purple", "p": "purple", "紫": "purple", "紫色": "purple",
	"a855f7": "purple", "9333ea": "purple",
}

// normalizeColor 把 AI 可能返回的各种颜色表达归一为标准 key。
func normalizeColor(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	return colorAliases[strings.TrimPrefix(c, "#")]
}

// moduleAliases 把 AI 返回的 module 表达归一为标准值。
var moduleAliases = map[string]string{
	"category": "category", "题型": "category", "分类": "category", "判断": "category",
	"rule": "rule", "规则": "rule", "技巧": "rule", "适用": "rule",
	"annotation": "annotation", "answer": "annotation", "答案": "annotation",
	"注释": "annotation", "思路": "annotation", "主旨": "annotation",
	"error": "error", "错误": "error", "转折": "error", "否定": "error",
	"info": "info", "关键": "info", "信息": "info", "条件": "info",
}

// normalizeModule 把 AI 返回的 module 表达归一为标准值。
func normalizeModule(m string) string {
	return moduleAliases[strings.ToLower(strings.TrimSpace(m))]
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
