package subject

import (
	"strings"
	"testing"
)

func TestParseSubject(t *testing.T) {
	md := `# 言语理解讲义（精简整理）

这是开篇概述，说明讲义结构。

## 第一章 片段阅读整体概述

### 1.1 言语考试大纲

大纲内容甲。

### 1.2 整体介绍与题型分类

题型分类内容乙。

## 第二章 中心理解之说理类文段

### 2.1 说理类文段解题逻辑总论

逻辑总论内容丙。
`
	sub, err := ParseSubject("言语理解", md)
	if err != nil {
		t.Fatalf("ParseSubject: %v", err)
	}
	if sub.ID != "言语理解" {
		t.Errorf("id = %q", sub.ID)
	}
	if len(sub.Tree) != 2 {
		t.Fatalf("tree roots = %d, want 2", len(sub.Tree))
	}
	if sub.Tree[0].Title != "第一章 片段阅读整体概述" {
		t.Errorf("root title = %q", sub.Tree[0].Title)
	}
	if len(sub.Tree[0].Children) != 2 {
		t.Errorf("children = %d, want 2", len(sub.Tree[0].Children))
	}
	if !strings.Contains(sub.Summary, "开篇概述") {
		t.Errorf("summary missing preface: %q", sub.Summary)
	}
	if !strings.Contains(sub.Tree[1].Children[0].Content, "逻辑总论内容丙") {
		t.Errorf("section content missing: %q", sub.Tree[1].Children[0].Content)
	}
	// IDs: 1, 1-1, 1-2, 2, 2-1
	if sub.Tree[1].ID != "2" || sub.Tree[1].Children[0].ID != "2-1" {
		t.Errorf("ids = %q / %q, want 2 / 2-1", sub.Tree[1].ID, sub.Tree[1].Children[0].ID)
	}
}

func TestParseSubjectEmpty(t *testing.T) {
	sub, err := ParseSubject("空讲义", "  \n\n  ")
	if err != nil {
		t.Fatalf("ParseSubject: %v", err)
	}
	if len(sub.Tree) != 0 {
		t.Errorf("tree = %d, want 0", len(sub.Tree))
	}
}

func TestHeadingSkipH1(t *testing.T) {
	if lvl, title, ok := heading("# 标题"); ok {
		t.Errorf("H1 should be skipped, got level=%d title=%q", lvl, title)
	}
	if lvl, title, ok := heading("## 二级"); !ok || lvl != 1 || title != "二级" {
		t.Errorf("H2: got %d %q %v", lvl, title, ok)
	}
	if lvl, _, ok := heading("#### 四级"); !ok || lvl != 3 {
		t.Errorf("H4: got %d %v", lvl, ok)
	}
	if _, _, ok := heading("普通文本"); ok {
		t.Errorf("plain text should not be a heading")
	}
}

func TestStoreNewMissingDir(t *testing.T) {
	s, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore empty dir: %v", err)
	}
	if len(s.List()) != 0 {
		t.Errorf("List = %d, want 0", len(s.List()))
	}
}
