package service

import (
	"encoding/json"
	"strings"
	"testing"

	"leans/model"
	"leans/subject"
)

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"plain", `{"a":1}`},
		{"fence", "```json\n{\"a\":1}\n```"},
		{"fence-no-lang", "```\n{\"a\":1}\n```"},
		{"surrounding-text", "好的，分析如下：\n{\"a\":1}\n希望对你有帮助"},
		{"comments", "{\n// 注释\n\"a\": 1,\n# 另一行注释\n\"b\": 2\n}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractJSON(c.in)
			if !strings.Contains(got, "a") {
				t.Errorf("extractJSON(%q) = %q, want object containing a", c.in, got)
			}
		})
	}
}

// TestSanitizeJSONBlock 覆盖"模型在 JSON 对象内部插入裸中文"的容错场景。
func TestSanitizeJSONBlock(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // 净化后应包含的子串（简单断言）
	}{
		{
			// 值后跟裸中文（本次实际报错的形态）
			"value-then-bare-text",
			"{\"answer\": \"B\", 意在说明主旨\n\"annotation\": \"x\"}",
			`"answer": "B"`,
		},
		{
			// 独立裸中文行
			"bare-text-line",
			"{\"a\": 1,\n这是多余说明\n\"b\": 2}",
			`"b": 2`,
		},
		{
			// 字符串内的中文不能误伤
			"keep-in-string",
			"{\"annotation\": \"这段文字意在强调重点\"}",
			"意在强调重点",
		},
		{
			// 合法 JSON 字面量 true/false 不能误伤
			"keep-literals",
			"{\"ok\": true, \"no\": false, \"n\": null}",
			"true",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeJSONBlock(c.in)
			if !strings.Contains(got, c.want) {
				t.Errorf("sanitizeJSONBlock(%q) = %q, want it to contain %q", c.in, got, c.want)
			}
			// 净化后的片段应能被 JSON 解析（作为对象的一部分）。
			if err := json.Unmarshal([]byte(got), &map[string]any{}); err != nil {
				t.Errorf("sanitizeJSONBlock(%q) not valid JSON: %v\ncleaned=%q", c.in, err, got)
			}
		})
	}
}

func TestParseAnalysisResponse(t *testing.T) {
	raw := "```json\n" + `{
  "type_judgment": {
    "category": "中心理解",
    "sub_category": "说理类-后对策结构",
    "basis": [
      {"text": "意在说明", "location": "题干", "explanation": "主旨设问词"}
    ]
  },
  "technique_judgment": {
    "rules": [
      {
        "name": "对策标志词识别",
        "section": "2.2",
        "usage": "看到必须/应该优先找对策",
        "example": "必须……",
        "marks": [{"text": "必须", "location": "选项B", "explanation": "对策标志"}]
      }
    ]
  },
  "answer": "B",
  "annotation": "先找主旨句……",
  "highlights": [
    {"text": "意在说明", "type": "设问词", "module": "category", "location": "题干", "explanation": "判断题型"},
    {"text": "必须", "type": "对策标志词", "module": "rule", "location": "选项B", "explanation": "命中规则"},
    {"text": "但是", "type": "转折词", "module": "error", "location": "题干", "explanation": "转折后是重点"}
  ]
}` + "\n```"
	res, err := ParseAnalysisResponse(raw)
	if err != nil {
		t.Fatalf("ParseAnalysisResponse: %v", err)
	}
	if res.Category != "中心理解" {
		t.Errorf("category = %q", res.Category)
	}
	if res.TypeJudgment == nil || res.TypeJudgment.Category != "中心理解" {
		t.Errorf("type_judgment mismatch: %+v", res.TypeJudgment)
	}
	if res.Answer != "B" {
		t.Errorf("answer = %q", res.Answer)
	}
	if len(res.Basis) != 1 || res.Basis[0].Text != "意在说明" {
		t.Errorf("basis mismatch: %+v", res.Basis)
	}
	if len(res.Rules) != 1 || len(res.Rules[0].Marks) != 1 {
		t.Errorf("rules mismatch: %+v", res.Rules)
	}
	if res.TechniqueJudgment == nil || len(res.TechniqueJudgment.Rules) != 1 {
		t.Errorf("technique_judgment mismatch: %+v", res.TechniqueJudgment)
	}
	byModule := map[string]int{}
	for _, h := range res.Highlights {
		byModule[h.Module]++
	}
	if byModule["category"] != 1 || byModule["rule"] != 1 || byModule["error"] != 1 {
		t.Errorf("module counts mismatch: %v", byModule)
	}
}

func TestParseAnalysisResponseLegacy(t *testing.T) {
	// 旧版字段应映射到新结构。
	raw := `{
  "category": "中心理解",
  "techniques": [{"name": "转折之后是重点", "section": "1.3", "description": "转折词后为主旨"}],
  "applicable": [{"rule_name": "对策标志词", "section": "2.2", "usage": "找对策", "example": "必须"}],
  "annotation": "注释",
  "highlights": [{"text": "但是", "type": "转折", "color": "red", "location": "题干", "explanation": "转折"}]
}`
	res, err := ParseAnalysisResponse(raw)
	if err != nil {
		t.Fatalf("ParseAnalysisResponse legacy: %v", err)
	}
	if len(res.Rules) != 1 || res.Rules[0].Name != "对策标志词" {
		t.Errorf("legacy applicable not mapped: %+v", res.Rules)
	}
	if len(res.Highlights) != 1 || res.Highlights[0].Color != "red" {
		t.Errorf("legacy highlight color not normalized: %+v", res.Highlights)
	}
	if res.TypeJudgment == nil || res.TypeJudgment.Category != "中心理解" {
		t.Errorf("legacy category not mapped to type_judgment: %+v", res.TypeJudgment)
	}
	if res.TechniqueJudgment == nil || len(res.TechniqueJudgment.Rules) != 1 {
		t.Errorf("legacy applicable not mapped to technique_judgment: %+v", res.TechniqueJudgment)
	}
}

func TestParseAnalysisResponseInvalid(t *testing.T) {
	if _, err := ParseAnalysisResponse("完全不是 JSON"); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestRepairJSONCommas 覆盖"模型输出 JSON 漏逗号"的容错场景。
func TestRepairJSONCommas(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			// 实际报错的形态：键值对之间漏逗号
			"missing-comma-pairs",
			`{"answer": "B" "annotation": "x"}`,
			`{"answer": "B", "annotation": "x"}`,
		},
		{
			"missing-comma-nested",
			"{\"type_judgment\": {\"category\": \"中心理解\" \"sub_category\": \"说理类\"}}",
			`{"type_judgment": {"category": "中心理解", "sub_category": "说理类"}}`,
		},
		{
			// 多位数字不能被拆散
			"valid-number-untouched",
			`{"n": 12, "arr": [1, 2]}`,
			`{"n": 12, "arr": [1, 2]}`,
		},
		{
			"missing-comma-numbers",
			`{"a": [1 2]}`,
			`{"a": [1, 2]}`,
		},
		{
			"missing-comma-object",
			`{"a": {"b": 1} "c": 2}`,
			`{"a": {"b": 1}, "c": 2}`,
		},
		{
			// 字符串内部的引号与逗号不能误伤
			"string-content-untouched",
			`{"s": "a\"b, c\" d", "t": "x"}`,
			`{"s": "a\"b, c\" d", "t": "x"}`,
		},
		{
			// true/false/null 结尾后漏逗号
			"missing-comma-literal",
			`{"a": true "b": false}`,
			`{"a": true, "b": false}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := repairJSONCommas(c.in)
			if got != c.want {
				t.Errorf("repairJSONCommas(%q) = %q, want %q", c.in, got, c.want)
			}
			var v any
			if err := json.Unmarshal([]byte(got), &v); err != nil {
				t.Errorf("repaired output not valid JSON: %v (%q)", err, got)
			}
		})
	}
}

func TestNormalizeModule(t *testing.T) {
	cases := map[string]string{
		"category": "category", "题型": "category", "分类": "category",
		"rule": "rule", "规则": "rule", "技巧": "rule",
		"annotation": "annotation", "答案": "annotation", "主旨": "annotation",
		"error": "error", "错误": "error", "转折": "error",
		"info": "info", "关键": "info", "": "",
	}
	for in, want := range cases {
		if got := normalizeModule(in); got != want {
			t.Errorf("normalizeModule(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeLocation(t *testing.T) {
	cases := map[string]string{
		"": "题干", "题干": "题干", "原文": "题干", "题目": "题干",
		"选项A": "选项A", "选项 a": "选项A", "A": "选项A", "选项B": "选项B",
		"选项C": "选项C", "选项D": "选项D", "b 选项": "选项B",
	}
	for in, want := range cases {
		if got := normalizeLocation(in); got != want {
			t.Errorf("normalizeLocation(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildAnalysisPrompt(t *testing.T) {
	hits := []subject.Hit{
		{Title: "第二章 / 2.1 说理类解题逻辑", Content: "对策标志词识别方法……"},
		{Title: "第一章 总体概述", Content: "讲义整体框架说明……"},
	}
	types := []string{"中心理解", "细节理解", "语句排序"}
	msgs := BuildAnalysisPrompt("言语理解", hits, types, "这段文字意在强调什么？", false, 6000)
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("system role missing")
	}
	body := msgs[1].Content
	if !strings.Contains(body, "说理类解题逻辑") {
		t.Errorf("prompt missing lecture sections: %s", body[:200])
	}
	if !strings.Contains(body, "无视觉约束") {
		t.Errorf("prompt missing no-vision constraint")
	}
	if !strings.Contains(body, "中心理解、细节理解、语句排序") {
		t.Errorf("prompt missing type catalog: %s", body[:200])
	}
	if !strings.Contains(body, "看不到任何图片、图表、截图、公式或版式") {
		t.Errorf("prompt no-vision constraint too weak")
	}
	if !strings.Contains(body, "这段文字意在强调什么？") {
		t.Errorf("prompt missing question")
	}
	// 概述章节只出现一次（整体概述槽位），不重复注入。
	if strings.Count(body, "讲义整体框架说明") != 1 {
		t.Errorf("overview duplicated: %s", body)
	}
	// 讲义注入量受预算限制。
	big := subject.Hit{Title: "第二章 对策题", Content: strings.Repeat("甲", 20000)}
	msgs = BuildAnalysisPrompt("言语理解", []subject.Hit{big}, nil, "题目", false, 6000)
	if len([]rune(msgs[1].Content)) > 10000 {
		t.Errorf("lecture injection not capped: %d chars", len([]rune(msgs[1].Content)))
	}
	if !strings.Contains(msgs[1].Content, "未提供题型清单") {
		t.Errorf("empty type catalog fallback missing")
	}
	// 题干不应超出注入上限（此处内容短，不会截断，但至少验证不 panic）。
	_ = model.AnalyzeResponse{}
}

func TestSyncJudgmentsFromNested(t *testing.T) {
	raw := `{
  "type_judgment": {"category": "数量关系", "sub_category": "工程问题", "basis": [{"text": "甲队", "location": "题干"}]},
  "technique_judgment": {"rules": [{"name": "赋值法", "section": "3.1", "marks": [{"text": "完成", "location": "选项A"}]}]}
}`
	res, err := ParseAnalysisResponse(raw)
	if err != nil {
		t.Fatalf("ParseAnalysisResponse: %v", err)
	}
	if res.Category != "数量关系" || len(res.Basis) != 1 {
		t.Errorf("nested type_judgment not synced to legacy: %+v", res)
	}
	if len(res.Rules) != 1 || len(res.Rules[0].Marks) != 1 {
		t.Errorf("nested technique_judgment not synced to legacy: %+v", res)
	}
	if res.TypeJudgment.Basis[0].Location != "题干" || res.Rules[0].Marks[0].Location != "选项A" {
		t.Errorf("location not normalized: %+v / %+v", res.TypeJudgment.Basis, res.Rules)
	}
}
