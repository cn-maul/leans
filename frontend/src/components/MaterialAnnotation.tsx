// MaterialAnnotation 是申论（主观题）标注视图。
//
// 与客观题不同，它有两个标注目标，用顶部小页签切换：
//   - 给定材料：把 result.highlights（采分点依据）按文本落到对应自然段。
//   - 我的作答：把 grading 里各采分点的 user_ref 句子按命中状态着色（仅当用户提交了作答）。
//
// 关键：不信任 AI 给的 location，改为按 highlight.text 在段落里的实际命中来归块
// （routeByText），材料段落序号常被报错，但 text 是从原文摘的。
import { useMemo, useState, type ReactNode } from 'react'
import { Highlighter } from 'lucide-react'
import type { Grading, Highlight } from '../types/analysis'
import { findMarkRanges, routeByText, MODULE_DOT, type Block } from '../lib/annotation'

interface Props {
  materialText: string
  answerText: string
  highlights: Highlight[]
  grading?: Grading | null
  active: boolean
}

type Target = 'material' | 'answer'

export default function MaterialAnnotation({ materialText, answerText, highlights, grading, active }: Props) {
  const hasAnswer = Boolean(answerText.trim())
  const [target, setTarget] = useState<Target>('material')

  const materialBlocks = useMemo(() => splitParagraphs(materialText, '材料段'), [materialText])
  const answerBlocks = useMemo(() => splitParagraphs(answerText, '作答'), [answerText])

  // 材料侧：按文本把 highlights 落到对应段落。
  const materialByBlock = useMemo(
    () => routeByText(materialBlocks, highlights, 'p1'),
    [materialBlocks, highlights],
  )
  // 作答侧：把 grading 的 user_ref 转成 highlight，按命中状态着色。
  const answerHighlights = useMemo(() => gradingToHighlights(grading), [grading])
  const answerByBlock = useMemo(
    () => routeByText(answerBlocks, answerHighlights, 'p1'),
    [answerBlocks, answerHighlights],
  )

  const showAnswer = hasAnswer && target === 'answer'
  const blocks = showAnswer ? answerBlocks : materialBlocks
  const groups = showAnswer ? answerByBlock : materialByBlock
  const emptyHint = showAnswer
    ? '批改完成后，你写的作答会在这里按采分点命中情况标色'
    : '分析完成后，给定材料中的采分点依据会在这里标色'

  const legend: Array<[string, string]> = showAnswer
    ? [
        ['rule', '已踩中'],
        ['info', '部分踩中'],
        ['annotation', '采分点表述'],
      ]
    : [
        ['category', '题型判断'],
        ['rule', '适用方法'],
        ['annotation', '答案/思路'],
        ['error', '错误/转折'],
        ['info', '关键信息'],
      ]

  return (
    <div className="flex max-h-full min-h-0 flex-col overflow-hidden rounded-card bg-surface shadow-card">
      <div className="flex h-10 shrink-0 items-center gap-3 border-b border-hairline px-4">
        <span className="inline-flex shrink-0 items-center gap-1.5 text-xs font-medium text-muted">
          <Highlighter className="h-3.5 w-3.5 text-body" aria-hidden="true" />
        </span>
        {/* 目标切换：仅在提供了作答时出现 */}
        {hasAnswer && (
          <div className="flex shrink-0 items-center gap-[3px] rounded-pill bg-fill p-[3px]">
            <Seg active={target === 'material'} onClick={() => setTarget('material')} label="给定材料" />
            <Seg active={target === 'answer'} onClick={() => setTarget('answer')} label="我的作答" />
          </div>
        )}
        <div className="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-x-3 gap-y-1">
          {legend.map(([m, label]) => (
            <Legend key={m} module={m} label={label} />
          ))}
        </div>
      </div>

      {active ? (
        <div className="min-h-0 flex-1 divide-y divide-hairline overflow-y-auto">
          {blocks.map((b) => (
            <section key={b.key} className="px-5 py-4">
              <div className="mb-2 inline-flex items-center rounded-chip bg-fill px-2 py-0.5 text-[11px] font-medium text-muted">
                {b.label}
              </div>
              <p className="text-[15px] leading-7 whitespace-pre-wrap text-body">
                {renderAnnotated(b.text, groups[b.key] || [])}
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
          <p className="text-xs text-quiet">{emptyHint}</p>
        </div>
      )}
    </div>
  )
}

// splitParagraphs 按空行把一段文本切成自然段块；忽略过短碎片。
function splitParagraphs(text: string, labelPrefix: string): Block[] {
  const paras = text
    .split(/\r?\n\s*\r?\n/)
    .map((p) => p.replace(/\r?\n/g, ' ').trim())
    .filter((p) => p.length > 0)
  if (paras.length === 0) {
    const only = text.trim()
    return [{ key: 'p1', label: `${labelPrefix} 1`, text: only }]
  }
  return paras.map((t, i) => ({ key: `p${i + 1}`, label: `${labelPrefix} ${i + 1}`, text: t }))
}

// gradingToHighlights 把批改采分点的 user_ref 句子转成作答侧标色项。
// hit → 绿(rule)、partial → 黄(info)；miss 无 user_ref 不产出色块。
function gradingToHighlights(grading?: Grading | null): Highlight[] {
  if (!grading?.points) return []
  const out: Highlight[] = []
  for (const p of grading.points) {
    if (!p.user_ref) continue
    out.push({
      text: p.user_ref,
      type: '采分点',
      module: p.status === 'partial' ? 'info' : 'rule',
      location: '作答',
      explanation: p.point,
    })
  }
  return out
}

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

function Seg({ active, onClick, label }: { active: boolean; onClick: () => void; label: string }) {
  return (
    <button
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex h-6 items-center rounded-pill px-2.5 text-[12px] transition-all duration-[250ms] ease-quart active:scale-[0.97] ${
        active ? 'bg-surface font-semibold text-ink shadow-[0_1px_3px_rgb(0_0_0/0.12)]' : 'font-medium text-muted hover:text-ink'
      }`}
    >
      {label}
    </button>
  )
}

function Legend({ module, label }: { module: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1 text-[11px] text-quiet">
      <span className={`inline-block h-2 w-2 rounded-full ${MODULE_DOT[module] || 'dot-info'}`} />
      {label}
    </span>
  )
}
