package subject

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Section is a node in the parsed lecture tree. Content holds the markdown
// body of this node (excluding child headings).
type Section struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Level    int        `json:"level"` // 1 = ##, 2 = ###, ...
	Content  string     `json:"content"`
	Children []*Section `json:"children,omitempty"`
}

// Subject is a parsed lecture file with its section tree.
type Subject struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Summary string     `json:"summary"` // text before the first ## heading
	Tree    []*Section `json:"tree"`

	// sections is a flattened list of all sections (deepest-first) used for
	// keyword retrieval. Not serialized.
	sections []*Section
}

// LoadSubject parses a single markdown lecture file.
func LoadSubject(path string) (*Subject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return ParseSubject(name, string(data))
}

// ParseSubject converts markdown text into a subject with a heading tree.
func ParseSubject(name, text string) (*Subject, error) {
	s := &Subject{
		ID:   name,
		Name: name,
	}

	lines := strings.Split(text, "\n")
	var stack []*Section // stack of open sections by level
	var pending strings.Builder

	flush := func() {
		if pending.Len() == 0 {
			return
		}
		body := strings.TrimSpace(pending.String())
		pending.Reset()
		if body == "" {
			return
		}
		if len(stack) == 0 {
			s.Summary += body + "\n"
			return
		}
		top := stack[len(stack)-1]
		if top.Content != "" {
			top.Content += "\n"
		}
		top.Content += body
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if level, title, ok := heading(trimmed); ok {
			flush()
			// Pop back to parent level.
			for len(stack) > 0 && stack[len(stack)-1].Level >= level {
				stack = stack[:len(stack)-1]
			}
			sec := &Section{
				Title: title,
				Level: level,
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, sec)
			} else {
				s.Tree = append(s.Tree, sec)
			}
			stack = append(stack, sec)
			continue
		}
		pending.WriteString(line)
		pending.WriteString("\n")
	}
	flush()

	s.assignIDs()
	s.flatten()
	return s, nil
}

// heading returns the heading level (## = 1, ### = 2, ...) and title text.
func heading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	// Count leading '#'.
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	rest := strings.TrimSpace(line[i:])
	if rest == "" {
		return 0, "", false
	}
	level := i - 1 // H1 (single #) is level 0 and skipped as the doc title
	if level < 1 {
		return 0, "", false
	}
	return level, rest, true
}

func (s *Subject) assignIDs() {
	// Each section gets an id like "2-1" derived from its position in the
	// flattened tree path. Children of the same parent are numbered in order.
	var walk func(parent *Section, prefix []string)
	walk = func(sec *Section, prefix []string) {
		idx := 1
		for _, child := range sec.Children {
			id := append(append([]string{}, prefix...), itoa(idx))
			child.ID = strings.Join(id, "-")
			walk(child, id)
			idx++
		}
	}
	for i, root := range s.Tree {
		root.ID = itoa(i + 1)
		walk(root, []string{root.ID})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func (s *Subject) flatten() {
	var walk func(sec *Section)
	walk = func(sec *Section) {
		s.sections = append(s.sections, sec)
		for _, child := range sec.Children {
			walk(child)
		}
	}
	for _, root := range s.Tree {
		walk(root)
	}
}

// Sections returns the flattened section list.
func (s *Subject) Sections() []*Section {
	return s.sections
}

// overviewKeywords 是判断"总体介绍"性质章节标题的统一关键词表，
// 检索与 prompt 构建共用，避免三处定义漂移。
var overviewKeywords = []string{"概述", "总论", "整体", "总体", "大纲", "导言", "引言"}

// IsOverviewTitle 判断章节标题是否属于"总体介绍"性质。
func IsOverviewTitle(title string) bool {
	for _, kw := range overviewKeywords {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

// Overview returns candidate "overview" sections whose titles mention common
// overall-introduction keywords, used to always inject a broad context.
func (s *Subject) Overview() []*Section {
	var out []*Section
	seen := map[string]bool{}
	for _, sec := range s.sections {
		if seen[sec.ID] {
			continue
		}
		if IsOverviewTitle(sec.Title) {
			out = append(out, sec)
			seen[sec.ID] = true
		}
	}
	return out
}

// TypeCatalog 从讲义标题生成紧凑的题型清单，供 prompt 限定 category/sub_category
// 的取值空间。只保留清洗后的章节标题，按章节树顺序去重，最多返回 limit 个。
func (s *Store) TypeCatalog(id string, limit int) []string {
	s.mu.RLock()
	sub, ok := s.subjects[id]
	s.mu.RUnlock()
	if !ok || sub == nil {
		return nil
	}
	if limit <= 0 {
		limit = 30
	}

	seen := map[string]bool{}
	var out []string
	for _, sec := range sub.sections {
		title := cleanTypeTitle(sec.Title)
		if title == "" || IsOverviewTitle(title) || seen[title] {
			continue
		}
		seen[title] = true
		out = append(out, title)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// typeNumberRe 匹配"第一章 / 第一节 / 1.1 / 1、 / 一、"这类序号前缀。
var typeNumberRe = regexp.MustCompile(`^(?:第[0-9一二三四五六七八九十百]+[章节篇部分]+|(?:\d+\.)*\d+|[一二三四五六七八九十]+)[、.．:：]?\s*`)

// cleanTypeTitle 去掉标题里的章节序号和冗余空白，保留题型名本身。
func cleanTypeTitle(title string) string {
	t := strings.TrimSpace(title)
	t = typeNumberRe.ReplaceAllString(t, "")
	t = strings.TrimSpace(t)
	if len([]rune(t)) > 20 {
		t = string([]rune(t)[:20])
	}
	return t
}
