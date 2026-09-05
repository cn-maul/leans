package subject

import (
	"sort"
	"strings"
	"unicode"
)

// Retrieval 控制一次检索返回多少、多长的内容。
type Retrieval struct {
	MaxSections int // 最多返回几个相关章节（含概述）
	MaxChars    int // 每个章节正文最多保留多少字符
	OverviewMax int // 最多注入几个概述章节
}

// Hit 是检索命中的一个章节。Section 指向缓存对象（只读），Content 是
// 截断后用于注入 prompt 的正文副本，避免污染缓存。
type Hit struct {
	Section *Section
	Score   int
	Title   string // 章节标题路径，如 "第二章 / 2.1 说理类解题逻辑"
	Content string // 截断后的章节正文
}

// Search 从题目中提取关键词，在 subject 中打分并返回最相关的章节。
// 概述类章节（标题含"概述/总论"等）始终优先注入，保证大上下文不缺失。
func (s *Store) Search(id, question string, r Retrieval) []Hit {
	s.mu.RLock()
	sub, ok := s.subjects[id]
	s.mu.RUnlock()
	if !ok || sub == nil {
		return nil
	}

	if r.MaxSections <= 0 {
		r.MaxSections = 4
	}
	if r.MaxChars <= 0 {
		r.MaxChars = 2000
	}
	if r.OverviewMax <= 0 {
		r.OverviewMax = 1
	}

	words := tokenize(question)
	if len(words) == 0 {
		return nil
	}

	var hits []Hit
	for _, sec := range sub.sections {
		score := scoreSection(sec, words)
		// 概述章节即使基础分 0 也保留（后面有加分），其余需要正分。
		if score <= 0 && !IsOverviewTitle(sec.Title) {
			continue
		}
		hits = append(hits, Hit{
			Section: sec,
			Score:   score,
		})
	}

	// 概述章节加分，保证整体介绍一定进入结果。
	for i := range hits {
		if IsOverviewTitle(hits[i].Section.Title) {
			hits[i].Score += 200
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})

	// 概述章节只保留前 OverviewMax 个，避免霸屏。
	var overviews, rest []Hit
	for _, h := range hits {
		if IsOverviewTitle(h.Section.Title) && len(overviews) < r.OverviewMax {
			overviews = append(overviews, h)
		} else if !IsOverviewTitle(h.Section.Title) {
			rest = append(rest, h)
		}
	}

	// 合并：概述优先 + 其余按分。
	var out []Hit
	out = append(out, overviews...)
	restLen := r.MaxSections - len(overviews)
	if restLen < 0 {
		restLen = 0
	}
	if restLen > len(rest) {
		restLen = len(rest)
	}
	out = append(out, rest[:restLen]...)

	// 构建标题路径 + 截断正文（写进 Hit.Content，不改缓存对象）。
	for i := range out {
		path := sectionPath(sub, out[i].Section)
		parts := append(path, out[i].Section.Title)
		out[i].Title = strings.Join(parts, " / ")
		if out[i].Section.Content != "" {
			content := out[i].Section.Content
			runes := []rune(content)
			if len(runes) > r.MaxChars {
				content = string(runes[:r.MaxChars]) + "\n……（章节已截断）"
			}
			out[i].Content = content
		}
	}
	return out
}

// scoreSection 计算一个章节与关键词的相关度：标题命中权重大于正文。
func scoreSection(sec *Section, words []string) int {
	title := sec.Title
	body := sec.Content
	score := 0
	for _, w := range words {
		if strings.Contains(title, w) {
			score += 3
		}
		if strings.Contains(body, w) {
			score++
		}
	}
	return score
}

// sectionPath 返回章节在树中的标题路径（不含自身）。
func sectionPath(sub *Subject, target *Section) []string {
	var find func(sec *Section, path []string) ([]string, bool)
	find = func(sec *Section, path []string) ([]string, bool) {
		if sec == target {
			return path, true
		}
		for _, c := range sec.Children {
			if p, ok := find(c, append(path, sec.Title)); ok {
				return p, true
			}
		}
		return nil, false
	}
	for _, root := range sub.Tree {
		if p, ok := find(root, nil); ok {
			return p
		}
	}
	return nil
}

// tokenize 把中文/英文题目切成检索词：
//   - 英文/数字按连续段切
//   - 中文按连续汉字段切，段长 > 4 时再滑窗产出 2-4 字词
//   - 过滤单字与停用词
func tokenize(s string) []string {
	var out []string
	var cjk, ascii strings.Builder
	lastIsCJK, lastIsASCII := false, false

	flush := func() {
		if ascii.Len() > 0 {
			w := strings.ToLower(ascii.String())
			ascii.Reset()
			if len([]rune(w)) >= 2 {
				out = append(out, w)
			}
		}
		if cjk.Len() > 0 {
			seg := cjk.String()
			cjk.Reset()
			runes := []rune(seg)
			if len(runes) <= 4 {
				out = append(out, seg)
			} else {
				for _, w := range windowWords(runes) {
					out = append(out, w)
				}
			}
		}
	}

	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if lastIsCJK {
				flush()
			}
			lastIsCJK, lastIsASCII = false, true
			ascii.WriteRune(r)
		case unicode.Is(unicode.Han, r):
			if lastIsASCII {
				flush()
			}
			lastIsCJK, lastIsASCII = true, false
			cjk.WriteRune(r)
		default:
			if lastIsCJK || lastIsASCII {
				flush()
			}
			lastIsCJK, lastIsASCII = false, false
		}
	}
	flush()

	// 去重 + 过滤停用词和过短词。
	seen := map[string]bool{}
	var out2 []string
	for _, w := range out {
		if len([]rune(w)) < 2 {
			continue
		}
		if stopwords[w] {
			continue
		}
		if !seen[w] {
			seen[w] = true
			out2 = append(out2, w)
		}
	}
	return out2
}

// windowWords 对长中文段做 2-4 字滑窗切词。
func windowWords(runes []rune) []string {
	var out []string
	for size := 4; size >= 2; size-- {
		for i := 0; i+size <= len(runes); i++ {
			out = append(out, string(runes[i:i+size]))
		}
	}
	return out
}

var stopwords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true, "和": true, "与": true,
	"或": true, "为": true, "对": true, "中": true, "有": true, "这": true,
	"那": true, "一": true, "不": true, "也": true, "就": true, "都": true,
	"而": true, "并": true, "及": true, "被": true, "把": true, "让": true,
	"从": true, "到": true, "向": true, "由": true, "于": true, "如": true,
	"如果": true, "但是": true, "所以": true, "因为": true, "同时": true,
	"以及": true, "什么": true, "如何": true, "哪个": true, "哪些": true,
	"正确": true, "错误": true, "属于": true, "在于": true, "说明": true,
	"主要": true, "以下": true, "下列": true, "关于": true, "说法": true,
	"表述": true, "选项": true, "题干": true, "题目": true, "答案": true,
	"分析": true, "不正确": true, "符合": true, "不符": true,
	"理解": true, "判断": true, "选择": true, "应该": true, "可以": true,
	"需要": true, "根据": true, "这段": true, "文字": true, "意在": true,
}
