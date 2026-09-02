package subject

import (
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	if got := tokenize(""); len(got) != 0 {
		t.Errorf("tokenize(\"\") = %v, want empty", got)
	}
	got := tokenize("abc")
	if len(got) != 1 || got[0] != "abc" {
		t.Errorf("tokenize(\"abc\") = %v, want [abc]", got)
	}
	// 长中文段滑窗出 2-4 字词。
	got = tokenize("对策标志词")
	if len(got) == 0 {
		t.Errorf("tokenize(对策标志词) empty")
	}
	joined := strings.Join(got, "|")
	for _, w := range []string{"对策", "标志词"} {
		if !strings.Contains(joined, w) {
			t.Errorf("tokenize(对策标志词) missing %q in %v", w, got)
		}
	}
}

func TestSearchOverviewBoost(t *testing.T) {
	md := `# 讲义
## 第一章 总体概述
整体介绍内容。
## 第二章 对策题
后对策结构：对策标志词识别方法详解。
## 第三章 无关内容
与题目无关的内容。
`
	sub, err := ParseSubject("t", md)
	if err != nil {
		t.Fatalf("ParseSubject: %v", err)
	}
	store := &Store{subjects: map[string]*Subject{"t": sub}}

	hits := store.Search("t", "后对策结构怎么用对策标志词？", Retrieval{MaxSections: 3, MaxChars: 200, OverviewMax: 2})
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	// 概述章节必须在前。
	if !strings.Contains(hits[0].Title, "总体概述") {
		t.Errorf("first hit should be overview, got %q", hits[0].Title)
	}
	// 对策章节应该命中（标题命中权重高）。
	found := false
	for _, h := range hits {
		if strings.Contains(h.Title, "对策题") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("对策题 section should be in hits: %v", hits)
	}
}

func TestSearchTruncation(t *testing.T) {
	md := "# 讲义\n## 对策章节\n" + strings.Repeat("甲", 500) + "\n"
	sub, _ := ParseSubject("t", md)
	store := &Store{subjects: map[string]*Subject{"t": sub}}

	hits := store.Search("t", "对策标志词", Retrieval{MaxSections: 3, MaxChars: 100, OverviewMax: 2})
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	if len([]rune(hits[0].Content)) > 110 {
		t.Errorf("content not truncated: %d chars", len([]rune(hits[0].Content)))
	}
}
