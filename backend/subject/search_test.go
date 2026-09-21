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

// TestChunkedBM25Retrieval 验证：一个长 section 会被切成多块，BM25 能把最相关的
// 那一块（而非整节）排到前面，标题路径带进 Hit.Title。
func TestChunkedBM25Retrieval(t *testing.T) {
	long := strings.Repeat("这是无关的填充内容用于撑长段落。", 40) // 远超 maxChunk，会被切开
	md := "# 讲义\n## 第三章 主旨概括\n" +
		long + "\n\n" +
		"意图判断题的核心是识别言外之意，对策标志词如\"应该、必须、需要\"往往引出作者的意图与主张，抓住这些词就能定位主旨句。\n"
	sub, err := ParseSubject("t", md)
	if err != nil {
		t.Fatalf("ParseSubject: %v", err)
	}
	if len(sub.chunks) < 2 {
		t.Fatalf("expected the long section to split into multiple chunks, got %d", len(sub.chunks))
	}
	store := &Store{subjects: map[string]*Subject{"t": sub}}

	hits := store.Search("t", "意图判断题如何用对策标志词定位主旨", Retrieval{MaxSections: 4, MaxChars: 2000, OverviewMax: 1})
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	// 命中的块正文应包含"意图判断/对策标志词"这段，而非无关填充块。
	best := hits[0].Content
	if !strings.Contains(best, "对策标志词") {
		t.Errorf("top chunk should contain the query-relevant sentence, got: %.60s", best)
	}
	// Title 应带出章节路径。
	if !strings.Contains(hits[0].Title, "主旨概括") {
		t.Errorf("hit title should carry section path, got %q", hits[0].Title)
	}
}
