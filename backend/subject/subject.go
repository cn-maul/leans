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

// Kind 区分科目的作答形态，决定 prompt 模板与标注/解析策略。
// Choice：客观题（言语理解等），有 A/B/C/D 选项，答案为字母，标注落在题干/选项。
// Subjective：主观题（申论），无选项，答案为一段话，标注落在给定材料与"我的作答"。
type Kind string

const (
	KindChoice     Kind = "choice"
	KindSubjective Kind = "subjective"
)

// subjectiveKeywords 命中即视为主观题科目。申论、数量关系的大题等均属此类。
var subjectiveKeywords = []string{"申论", "作文", "大作文"}

// KindFor 依据科目名推断种类。默认客观题（保持既有科目行为不变）。
func KindFor(name string) Kind {
	for _, kw := range subjectiveKeywords {
		if strings.Contains(name, kw) {
			return KindSubjective
		}
	}
	return KindChoice
}

// Subject is a parsed lecture file with its section tree.
type Subject struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Summary string     `json:"summary"` // text before the first ## heading
	Tree    []*Section `json:"tree"`
	Kind    Kind       `json:"kind"` // choice | subjective，决定 AI 分析策略

	// sections is a flattened list of all sections (deepest-first) used for
	// keyword retrieval. Not serialized.
	sections []*Section

	// chunks 是把每个 section 的正文按段落聚合成的检索小块（~minChunk..maxChunk 字）。
	// 相比整节注入，小块让"注入更紧、召回更全"，是省 token 的主手段。Not serialized.
	chunks []*Chunk

	// BM25 语料统计（加载时预计算，检索时零外部依赖）。
	chunkTF  []map[string]int // 每块的词频（与 chunks 下标对齐）
	chunkLen []int            // 每块的词总数（文档长度）
	df       map[string]int   // 词 → 出现它的块数
	avgdl    float64          // 平均块长
}

// Chunk 是一个检索用小块：隶属某个 Section，携带预计算的标题路径。
type Chunk struct {
	Section    *Section
	Seq        int    // 该 Section 内第几块（0-based）
	Text       string // 块正文
	Path       string // 标题路径，如 "第二章 归纳概括题 / 一、概括能力"
	IsOverview bool   // 是否属于"总体介绍"章节
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
		Kind: KindFor(name),
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
	s.buildChunks()
	return s, nil
}

// 分块长度目标：段落尽量聚到 ~minChunk 字以上，超过 maxChunk 的长段按句子切开。
const (
	minChunk = 120
	maxChunk = 600
)

// buildChunks 把每个 section 的正文切成检索小块，并预计算 BM25 语料统计。
// 在 flatten() 之后调用；失败不致命（无块时检索退化为空结果）。
func (s *Subject) buildChunks() {
	var walk func(sec *Section, path []string)
	walk = func(sec *Section, path []string) {
		if sec.Content != "" {
			full := append(append([]string{}, path...), sec.Title)
			pathStr := strings.Join(full, " / ")
			isOv := IsOverviewTitle(sec.Title)
			for i, text := range splitChunks(sec.Content) {
				s.chunks = append(s.chunks, &Chunk{
					Section:    sec,
					Seq:        i,
					Text:       text,
					Path:       pathStr,
					IsOverview: isOv,
				})
			}
		}
		childPath := append(append([]string{}, path...), sec.Title)
		for _, c := range sec.Children {
			walk(c, childPath)
		}
	}
	for _, root := range s.Tree {
		walk(root, nil)
	}
	if len(s.chunks) == 0 {
		return
	}

	// BM25 语料统计。
	s.chunkTF = make([]map[string]int, len(s.chunks))
	s.chunkLen = make([]int, len(s.chunks))
	s.df = map[string]int{}
	total := 0
	for i, c := range s.chunks {
		tf := termFreqs(c.Text)
		s.chunkTF[i] = tf
		n := 0
		for _, v := range tf {
			n += v
		}
		s.chunkLen[i] = n
		total += n
		for term := range tf {
			s.df[term]++
		}
	}
	s.avgdl = float64(total) / float64(len(s.chunks))
}

// splitChunks 把一段 markdown 正文按段落聚合成 ~minChunk..maxChunk 字的小块。
// 短段落向后合并直到达到 minChunk；超过 maxChunk 的段落按句子边界再切。
func splitChunks(content string) []string {
	paras := paragraphs(content)
	var out []string
	var buf strings.Builder
	bufLen := 0

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, strings.TrimSpace(buf.String()))
		buf.Reset()
		bufLen = 0
	}

	for _, p := range paras {
		pl := len([]rune(p))
		// 超长段落：先把缓冲 flush，再单独按句切。
		if pl > maxChunk {
			flush()
			for _, piece := range splitBySentences(p) {
				out = append(out, piece)
			}
			continue
		}
		if bufLen+pl > maxChunk {
			flush()
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(p)
		bufLen += pl
		// 够一个最小块就先落地，保持小块粒度。
		if bufLen >= minChunk {
			flush()
		}
	}
	flush()
	return out
}

// paragraphs 按空行切段，过滤纯 markdown 标题行与空白段。
func paragraphs(content string) []string {
	lines := strings.Split(content, "\n")
	var paras []string
	var cur strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if cur.Len() > 0 {
				paras = append(paras, strings.TrimSpace(cur.String()))
				cur.Reset()
			}
			continue
		}
		if cur.Len() > 0 {
			cur.WriteString("\n")
		}
		cur.WriteString(line)
	}
	if cur.Len() > 0 {
		paras = append(paras, strings.TrimSpace(cur.String()))
	}
	// 过滤只含标题的段（理论上 section.Content 已排除标题，双保险）。
	var out []string
	for _, p := range paras {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// splitBySentences 把一个超长段落按中文/英文句末标点切成不超过 maxChunk 的小块。
func splitBySentences(p string) []string {
	runes := []rune(p)
	var out []string
	start := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		isEnd := r == '。' || r == '！' || r == '？' || r == '；' || r == '.' || r == '!' || r == '?' || r == ';' || r == '\n'
		if !isEnd {
			continue
		}
		// 句末且已累计超过 maxChunk 才落地，避免碎片。
		if i+1-start >= maxChunk {
			out = append(out, strings.TrimSpace(string(runes[start:i+1])))
			start = i + 1
		}
	}
	if start < len(runes) {
		tail := strings.TrimSpace(string(runes[start:]))
		if tail != "" {
			out = append(out, tail)
		}
	}
	if len(out) == 0 {
		out = append(out, p)
	}
	return out
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
