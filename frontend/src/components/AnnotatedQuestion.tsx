// AnnotatedQuestion renders the original (客观题) question text with AI highlight
// annotations applied directly onto the source text (in-place colored marks).
//
// 分块方式：题干 + 选项A–H。highlight 归属哪块按 location 决定。
// 匹配、颜色、图例等通用逻辑来自 lib/annotation，与申论材料视图共用。
import { useMemo, type ReactNode } from 'react'
import { Highlighter } from 'lucide-react'
import type { Highlight } from '../types/analysis'
import { findMarkRanges, groupByLocation, MODULE_DOT, type Block } from '../lib/annotation'

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
    <div className="flex max-h-full min-h-0 flex-col overflow-hidden rounded-card bg-surface shadow-card">
      <div className="flex h-10 shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-hairline px-4">
        <span className="inline-flex items-center gap-1.5 text-xs font-medium text-muted">
          <Highlighter className="h-3.5 w-3.5 text-body" aria-hidden="true" />
          标注视图
        </span>
        {legend.map(([m, label]) => (
          <Legend key={m} module={m} label={label} />
        ))}
      </div>

      {active ? (
        <div className="min-h-0 flex-1 divide-y divide-hairline overflow-y-auto">
          {blocks.map((b) => (
            <section key={b.key} className="px-5 py-4">
              <div className="mb-2 inline-flex items-center rounded-chip bg-fill px-2 py-0.5 text-[11px] font-medium text-muted">
                {b.label}
              </div>
              <p className="text-[15px] leading-7 whitespace-pre-wrap text-body">
                {renderAnnotated(b.text, byLoc[b.key] || [])}
              </p>
            </section>
          ))}
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-1.5 p-8 text-center">
          <div className="flex h-11 w-11 items-center justify-center rounded-thumb border border-dashed border-hint text-ghost">
            <Highlighter className="h-5 w-5" aria-hidden="true" />
          </div>
          <p className="mt-1 text-sm text-muted">等待分析结果</p>
          <p className="text-xs text-quiet">分析完成后，题干与选项将在这里分区标注</p>
        </div>
      )}
    </div>
  )
}

// ---- 客观题分块：题干 + 选项 A–H ----

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

// renderAnnotated 把一段文本与其归属的 highlight 渲染为带色 <mark>。
function renderAnnotated(text: string, hs: Highlight[]): ReactNode[] {
  const ranges = findMarkRanges(text, hs)
  if (ranges.length === 0) return [text]

  const out: ReactNode[] = []
  let pos = 0
  for (const r of ranges) {
    if (r.start > pos) out.push(text.slice(pos, r.start))
    out.push(
      <mark key={r.start} className={r.cls} title={r.tip}>
        {text.slice(r.start, r.end + 1)}
      </mark>,
    )
    pos = r.end + 1
  }
  if (pos < text.length) out.push(text.slice(pos))
  return out
}

function Legend({ module, label }: { module: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1 text-[11px] text-quiet">
      <span className={`inline-block h-2 w-2 rounded-full ${MODULE_DOT[module] || 'dot-info'}`} />
      {label}
    </span>
  )
}
