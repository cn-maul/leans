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
  "category": "中心理解",
  "sub_category": "说理类-后对策结构",
  "answer": "B",
  "basis": [
    {"text": "意在说明", "location": "题干", "explanation": "主旨设问词"}
  ],
  "rules": [
    {
      "name": "对策标志词识别",
      "section": "2.2",
      "usage": "看到必须/应该优先找对策",
      "example": "必须……",
      "marks": [{"text": "必须", "location": "选项B", "explanation": "对策标志"}]
    }
  ],
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
	if res.Answer != "B" {
		t.Errorf("answer = %q", res.Answer)
	}
	if len(res.Basis) != 1 || res.Basis[0].Text != "意在说明" {
		t.Errorf("basis mismatch: %+v", res.Basis)
	}
	if len(res.Rules) != 1 || len(res.Rules[0].Marks) != 1 {
		t.Errorf("rules mismatch: %+v", res.Rules)
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
}

func TestParseAnalysisResponseInvalid(t *testing.T) {
	if _, err := ParseAnalysisResponse("完全不是 JSON"); err == nil {
		t.Error("expected error for invalid JSON")
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
	msgs := BuildAnalysisPrompt("言语理解", hits, "这段文字意在强调什么？", false, 6000)
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
	if !strings.Contains(body, "这段文字意在强调什么？") {
		t.Errorf("prompt missing question")
	}
	// 概述章节只出现一次（整体概述槽位），不重复注入。
	if strings.Count(body, "讲义整体框架说明") != 1 {
		t.Errorf("overview duplicated: %s", body)
	}
	// 讲义注入量受预算限制。
	big := subject.Hit{Title: "第二章 对策题", Content: strings.Repeat("甲", 20000)}
	msgs = BuildAnalysisPrompt("言语理解", []subject.Hit{big}, "题目", false, 6000)
	if len([]rune(msgs[1].Content)) > 10000 {
		t.Errorf("lecture injection not capped: %d chars", len([]rune(msgs[1].Content)))
	}
	// 题干不应超出注入上限（此处内容短，不会截断，但至少验证不 panic）。
	_ = model.AnalyzeResponse{}
}
