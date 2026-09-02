package subject

import (
	"os"
	"path/filepath"
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

// Overview returns candidate "overview" sections whose titles mention common
// overall-introduction keywords, used to always inject a broad context.
func (s *Subject) Overview() []*Section {
	var out []*Section
	seen := map[string]bool{}
	for _, sec := range s.sections {
		if seen[sec.ID] {
			continue
		}
		t := sec.Title
		if strings.Contains(t, "概述") || strings.Contains(t, "总论") ||
			strings.Contains(t, "整体") || strings.Contains(t, "大纲") ||
			strings.Contains(t, "总体") {
			out = append(out, sec)
			seen[sec.ID] = true
		}
	}
	return out
}
