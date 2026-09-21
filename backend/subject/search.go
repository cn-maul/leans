package subject

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Retrieval 控制一次检索返回多少、多长的内容。
type Retrieval struct {
	MaxSections int // 最多返回几个检索块（含概述）
	MaxChars    int // 每个块正文最多保留多少字符
	OverviewMax int // 最多注入几个概述块
}

// Hit 是检索命中的一个内容块。Section 指向缓存对象（只读），Content 是
// 截断后用于注入 prompt 的正文副本，避免污染缓存。
type Hit struct {
	Section *Section
	Score   int
	Title   string // 检索块标题，如 "第二章 / 2.1 说理类解题逻辑 ·块3"
	Content string // 截断后的块正文
}

// BM25 参数：k1 控制词频饱和，b 控制长度归一化强度。
const (
	bm25K1 = 1.5
	bm25B  = 0.75
	// pathBonus 是查询词命中标题路径时的加分，远小于正文 BM25，只做同分排序倾斜。
	pathBonus = 0.8
)

// Search 用 BM25 在细粒度检索块中打分，返回最相关的内容块。
// 概述块始终优先注入若干，保证大上下文不缺失。返回类型仍是 []Hit，
// 使 prompt/analyzer 等下游接口无需改动。
func (s *Store) Search(id, question string, r Retrieval) []Hit {
	s.mu.RLock()
	sub, ok := s.subjects[id]
	s.mu.RUnlock()
	if !ok || sub == nil || len(sub.chunks) == 0 {
		return nil
	}

	if r.MaxSections <= 0 {
		r.MaxSections = 6
	}
	if r.MaxChars <= 0 {
		r.MaxChars = 1200
	}
	if r.OverviewMax <= 0 {
		r.OverviewMax = 1
	}

	words := tokenize(question)
	if len(words) == 0 {
		return nil
	}

	N := float64(len(sub.chunks))
	type scored struct {
		chunk *Chunk
		score float64
	}

	var overviews, rest []scored
	for i, c := range sub.chunks {
		score := bm25Score(sub, i, words, N)
		// 标题路径命中查询词时给少量加分：让"标题相关但正文无该词"的块也可召回。
		lpath := strings.ToLower(c.Path)
		for _, w := range words {
			if strings.Contains(lpath, w) {
				score += pathBonus
			}
		}
		if score <= 0 && !c.IsOverview {
			continue
		}
		item := scored{chunk: c, score: score}
		if c.IsOverview {
			overviews = append(overviews, item)
		} else {
			rest = append(rest, item)
		}
	}

	byScore := func(a, b scored) bool { return a.score > b.score }
	sort.SliceStable(overviews, func(i, j int) bool { return byScore(overviews[i], overviews[j]) })
	sort.SliceStable(rest, func(i, j int) bool { return byScore(rest[i], rest[j]) })

	// 合并：概述优先占 OverviewMax 个，其余按 BM25 分数补足到 MaxSections。
	var picked []scored
	if len(overviews) > r.OverviewMax {
		overviews = overviews[:r.OverviewMax]
	}
	picked = append(picked, overviews...)
	slots := r.MaxSections - len(picked)
	if slots < 0 {
		slots = 0
	}
	if slots > len(rest) {
		slots = len(rest)
	}
	picked = append(picked, rest[:slots]...)

	// 转成 Hit：截断正文、拼标题。
	out := make([]Hit, 0, len(picked))
	for _, sc := range picked {
		c := sc.chunk
		title := c.Path
		if c.Seq > 0 {
			title += " ·块" + itoa(c.Seq+1)
		}
		content := c.Text
		if runec := []rune(content); len(runec) > r.MaxChars {
			content = string(runec[:r.MaxChars]) + "\n……（内容已截断）"
		}
		out = append(out, Hit{
			Section: c.Section,
			Score:   int(math.Round(sc.score)),
			Title:   title,
			Content: content,
		})
	}
	return out
}

// bm25Score 计算第 idx 个块对查询词的 BM25 得分。
func bm25Score(sub *Subject, idx int, words []string, N float64) float64 {
	tf := sub.chunkTF[idx]
	dl := float64(sub.chunkLen[idx])
	avgdl := sub.avgdl
	if avgdl <= 0 {
		avgdl = 1
	}
	score := 0.0
	for _, w := range words {
		f, ok := tf[w]
		if !ok || f == 0 {
			continue
		}
		df := float64(sub.df[w])
		idf := math.Log(1 + (N-df+0.5)/(df+0.5))
		tfNorm := float64(f) * (bm25K1 + 1)
		tfNorm /= float64(f) + bm25K1*(1-bm25B+bm25B*dl/avgdl)
		score += idf * tfNorm
	}
	return score
}

// termFreqs 统计一段文本的词频（不去重），供 BM25 建索引使用。
// 与 tokenize 的区别：tokenize 会去重供查询用，这里保留重复以体现真实词频。
func termFreqs(s string) map[string]int {
	tokens := tokenizeRaw(s)
	tf := make(map[string]int, len(tokens))
	for _, w := range tokens {
		tf[w]++
	}
	return tf
}

// tokenize 把中文/英文题目切成检索词（去重后，供查询使用）。
func tokenize(s string) []string {
	raw := tokenizeRaw(s)
	seen := map[string]bool{}
	var out []string
	for _, w := range raw {
		if stopwords[w] {
			continue
		}
		if !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

// tokenizeRaw 切词但保留重复、不过滤停用词（供建索引统计词频）。
func tokenizeRaw(s string) []string {
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

	// 过滤过短词；不去重、不过滤停用词（停用词过滤只在查询侧做）。
	var out2 []string
	for _, w := range out {
		if len([]rune(w)) < 2 {
			continue
		}
		out2 = append(out2, w)
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
