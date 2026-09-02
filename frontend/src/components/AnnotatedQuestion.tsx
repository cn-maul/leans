// AnnotatedQuestion renders the original question text with AI highlight
// annotations applied directly onto the source text (in-place colored marks).
//
// Colors follow the module the highlight belongs to (category/rule/annotation/
// error/info) so that marks on the left match the three panels on the right.
//
// Matching is tolerant: AI sometimes rewrites the highlighted phrase with
// different punctuation or full/half-width chars. We normalize both sides
// (full-width -> half-width, lowercase, whitespace collapse) and fall back to
// a punctuation-stripped substring match before giving up.
import { useMemo, type ReactNode } from 'react'
import type { Highlight } from '../types/analysis'
import { MODULE_COLOR, MODULE_LABEL } from '../types/analysis'

interface Props {
  question: string
  highlights: Highlight[]
}

export default function AnnotatedQuestion({ question, highlights }: Props) {
  const blocks = useMemo(() => parseBlocks(question), [question])
  const byLoc = useMemo(() => groupByLocation(highlights), [highlights])

  const legend: Array<[string, string]> = [
    ['category', '题型判断'],
    ['rule', '适用规则'],
    ['annotation', '答案/思路'],
    ['error', '错误/转折'],
    ['info', '关键信息'],
  ]

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-5 shadow-sm space-y-4">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs font-medium text-gray-400 dark:text-gray-500">标注图例</span>
        {legend.map(([m, label]) => (
          <Legend key={m} color={MODULE_COLOR[m]} label={`${label} · ${MODULE_LABEL[m]}`} />
        ))}
      </div>

      {blocks.map((b) => (
        <section key={b.key} className="rounded-lg border border-gray-100 dark:border-gray-800 p-4">
          <div className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wide mb-2">
            {b.label}
          </div>
          <p className="text-[15px] leading-relaxed text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
            {annotate(b.text, byLoc[b.key] || [])}
          </p>
        </section>
      ))}
    </div>
  )
}

// ---- parsing & annotation helpers ----

interface Block {
  key: string
  label: string
  text: string
}

// Split question text into the stem (题干) plus each option.
function parseBlocks(q: string): Block[] {
  const lines = q.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  const blocks: Block[] = []
  let stem: string[] = []
  const optionRe = /^([A-H])[.、．:：）)‑]\s*/

  for (const line of lines) {
    const m = line.match(optionRe)
    if (m) {
      if (blocks.length === 0 && stem.length) {
        blocks.push({ key: 'stem', label: '题干', text: stem.join(' ') })
      }
      blocks.push({ key: m[1], label: `选项 ${m[1]}`, text: line.replace(optionRe, '').trim() })
    } else if (blocks.length === 0) {
      stem.push(line)
    } else {
      blocks[blocks.length - 1].text += ' ' + line
    }
  }
  if (blocks.length === 0) {
    blocks.push({ key: 'stem', label: '题干', text: q.trim() })
  }
  return blocks
}

// Normalize a highlight location string to a block key ("A", "B", "stem", ...).
function normalizeLoc(loc: string): string {
  const s = (loc || '').trim()
  const m = s.match(/[A-H]/i)
  if (m) return m[0].toUpperCase()
  return 'stem'
}

function groupByLocation(hs: Highlight[]): Record<string, Highlight[]> {
  const groups: Record<string, Highlight[]> = {}
  for (const h of hs) {
    const key = normalizeLoc(h.location)
    ;(groups[key] ||= []).push(h)
  }
  return groups
}

interface MarkRange {
  start: number
  end: number
  color: string
  tip: string
}

// ---- tolerant matching ----

// Normalize a single char: full-width -> half-width, lowercase.
function normChar(ch: string): string {
  const code = ch.codePointAt(0) ?? 0
  if (code >= 0xff01 && code <= 0xff5e) {
    return String.fromCodePoint(code - 0xfee0).toLowerCase()
  }
  if (code === 0x3000) return ' '
  return ch.toLowerCase()
}

// Returns the normalized text plus a map from normalized index -> original index.
function normalizeWithMap(text: string): { norm: string; map: number[] } {
  const out: string[] = []
  const map: number[] = []
  for (let i = 0; i < text.length; i++) {
    out.push(normChar(text[i]))
    map.push(i)
  }
  return { norm: out.join(''), map }
}

// Compact form: strip whitespace + punctuation from normalized text, keeping a
// map from compact index -> normalized index.
function compactForm(text: string): { text: string; map: number[] } {
  const out: string[] = []
  const map: number[] = []
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (/\s|[\u3000-\u303f\uff00-\uffef，。、；：？！,.!?;:'"“”‘’（）()【】\[\]<>《》\-—…·]/.test(ch)) {
      continue
    }
    out.push(ch)
    map.push(i)
  }
  return { text: out.join(''), map }
}

// annotate matches highlight phrases against a block of original text.
function annotate(text: string, hs: Highlight[]): ReactNode[] {
  if (hs.length === 0) return [text]

  const { norm, map } = normalizeWithMap(text)
  const compact = compactForm(norm)
  const ranges: MarkRange[] = []

  for (const h of hs) {
    if (!h.text) continue
    const needleNorm = normalizeWithMap(h.text).norm.trim()

    // Tier 1: normalized exact substring.
    let idx = norm.indexOf(needleNorm)
    if (idx !== -1) {
      pushRange(ranges, map[idx], map[idx + needleNorm.length - 1], h)
      continue
    }

    // Tier 2: punctuation/whitespace stripped, substring on compact forms.
    const needleCompact = compactForm(needleNorm).text
    if (needleCompact) {
      const ci = compact.text.indexOf(needleCompact)
      if (ci !== -1) {
        const startN = compact.map[ci]
        const endN = compact.map[ci + needleCompact.length - 1]
        pushRange(ranges, map[startN], map[endN], h)
        continue
      }
    }
    // Tier 3: fall back to highlighting nothing (silently skip).
  }

  if (ranges.length === 0) return [text]

  ranges.sort((a, b) => a.start - b.start)
  const merged: MarkRange[] = []
  for (const r of ranges) {
    const last = merged[merged.length - 1]
    if (last && r.start <= last.end) {
      if (r.end > last.end) last.end = r.end
    } else {
      merged.push({ ...r })
    }
  }

  const out: ReactNode[] = []
  let pos = 0
  for (const r of merged) {
    if (r.start > pos) out.push(text.slice(pos, r.start))
    out.push(
      <mark key={r.start} className={markClass(r.color)} title={r.tip}>
        {text.slice(r.start, r.end + 1)}
      </mark>,
    )
    pos = r.end + 1
  }
  if (pos < text.length) out.push(text.slice(pos))
  return out
}

function pushRange(ranges: MarkRange[], start: number, end: number, h: Highlight) {
  if (start < 0 || end < start) return
  // 颜色以 module 为准（与右侧模块一致），旧数据回退到 color 字段。
  const color = MODULE_COLOR[h.module || ''] || h.color || 'blue'
  ranges.push({ start, end, color, tip: h.type + (h.explanation ? '：' + h.explanation : '') })
}

const MARK_CLASS: Record<string, string> = {
  green: 'bg-green-200 text-green-900 rounded px-0.5 dark:bg-green-900/60 dark:text-green-100',
  red: 'bg-red-200 text-red-900 rounded px-0.5 dark:bg-red-900/60 dark:text-red-100',
  blue: 'bg-blue-200 text-blue-900 rounded px-0.5 dark:bg-blue-900/60 dark:text-blue-100',
  yellow: 'bg-yellow-200 text-yellow-900 rounded px-0.5 dark:bg-yellow-900/60 dark:text-yellow-100',
  purple: 'bg-purple-200 text-purple-900 rounded px-0.5 dark:bg-purple-900/60 dark:text-purple-100',
}

const DOT_CLASS: Record<string, string> = {
  green: 'bg-green-500',
  red: 'bg-red-500',
  blue: 'bg-blue-500',
  yellow: 'bg-yellow-500',
  purple: 'bg-purple-500',
}

function markClass(color: string): string {
  return MARK_CLASS[color] || MARK_CLASS.blue
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
      <span className={`inline-block w-2.5 h-2.5 rounded-full ${DOT_CLASS[color] || 'bg-gray-400'}`} />
      {label}
    </span>
  )
}
