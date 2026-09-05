// AnnotatedQuestion renders the original question text with AI highlight
// annotations applied directly onto the source text (in-place colored marks).
//
// Colors follow the module the highlight belongs to (category/rule/annotation/
// error/info) so that marks on the left match the modules on the right.
// annotation 模块使用墨色（ink），整体保持极简单色科技风。
//
// Matching is tolerant: AI sometimes rewrites the highlighted phrase with
// different punctuation or full/half-width chars. We normalize both sides
// (full-width -> half-width, lowercase, whitespace collapse) and fall back to
// a punctuation-stripped substring match before giving up.
import { useMemo, type ReactNode } from 'react'
import { Highlighter } from 'lucide-react'
import type { Highlight } from '../types/analysis'
import { MODULE_COLOR, LEGACY_COLOR } from '../types/analysis'

interface Props {
  question: string
  highlights: Highlight[]
  // active 表示当前已有分析结果；未分析时显示占位而不是重复展示题目原文。
  active: boolean
}

export default function AnnotatedQuestion({ question, highlights, active }: Props) {
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
    <div className="flex min-h-0 flex-1 animate-fade-up flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex h-10 shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-zinc-100 px-4 dark:border-zinc-800">
        <span className="inline-flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          <Highlighter className="h-3.5 w-3.5 text-zinc-700 dark:text-zinc-300" aria-hidden="true" />
          标注视图
        </span>
        {legend.map(([m, label]) => (
          <Legend key={m} color={MODULE_COLOR[m]} label={label} />
        ))}
      </div>

      {active ? (
        <div className="min-h-0 flex-1 divide-y divide-zinc-100 overflow-y-auto dark:divide-zinc-800">
          {blocks.map((b, i) => (
            <section key={b.key} className="animate-fade-up px-5 py-4" style={{ animationDelay: `${i * 40}ms` }}>
              <div className="mb-2 inline-flex items-center rounded-md bg-zinc-100 px-2 py-0.5 text-[11px] font-medium text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">
                {b.label}
              </div>
              <p className="text-[15px] leading-7 whitespace-pre-wrap text-zinc-800 dark:text-zinc-200">
                {annotate(b.text, byLoc[b.key] || [])}
              </p>
            </section>
          ))}
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-1.5 p-8 text-center">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-dashed border-zinc-300 text-zinc-300 dark:border-zinc-700 dark:text-zinc-600">
            <Highlighter className="h-5 w-5" aria-hidden="true" />
          </div>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">等待分析结果</p>
          <p className="text-xs text-zinc-400 dark:text-zinc-500">
            分析完成后，题干与选项将在这里分区标注
          </p>
        </div>
      )}
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
    if (/\s|[\u3000-\u303f\uff00-\uffef，。、；：？！,.!?;:'"“”‘’（）()【】[]<>《》\-—…·]/.test(ch)) {
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
  // 颜色以 module 为准（与右侧模块一致），旧数据的 color 兜底（紫 → 墨色）。
  const color = MODULE_COLOR[h.module || ''] || LEGACY_COLOR[h.color || ''] || h.color || 'blue'
  ranges.push({ start, end, color, tip: h.type + (h.explanation ? '：' + h.explanation : '') })
}

const MARK_CLASS: Record<string, string> = {
  green:
    'rounded-[4px] bg-emerald-100 px-0.5 text-emerald-900 dark:bg-emerald-400/15 dark:text-emerald-300',
  red: 'rounded-[4px] bg-rose-100 px-0.5 text-rose-900 dark:bg-rose-400/15 dark:text-rose-300',
  blue: 'rounded-[4px] bg-sky-100 px-0.5 text-sky-900 dark:bg-sky-400/15 dark:text-sky-300',
  yellow:
    'rounded-[4px] bg-amber-100 px-0.5 text-amber-900 dark:bg-amber-400/15 dark:text-amber-300',
  ink: 'rounded-[4px] bg-zinc-900 px-0.5 text-white dark:bg-zinc-100 dark:text-zinc-900',
  // 旧数据兼容
  purple: 'rounded-[4px] bg-zinc-900 px-0.5 text-white dark:bg-zinc-100 dark:text-zinc-900',
}

const DOT_CLASS: Record<string, string> = {
  green: 'bg-emerald-500',
  red: 'bg-rose-500',
  blue: 'bg-sky-500',
  yellow: 'bg-amber-500',
  ink: 'bg-zinc-900 dark:bg-zinc-100',
  purple: 'bg-zinc-900 dark:bg-zinc-100',
}

function markClass(color: string): string {
  return MARK_CLASS[color] || MARK_CLASS.blue
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1 text-[11px] text-zinc-400 dark:text-zinc-500">
      <span
        className={`inline-block h-2 w-2 rounded-full ${DOT_CLASS[color] || 'bg-zinc-400'}`}
      />
      {label}
    </span>
  )
}
