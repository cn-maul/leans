// ShenlunPage 是申论（主观题）独立工作区。
// 复用后端同一条 /analyze/stream 与整条流式、历史、统计链路，只在视图与输入上做主观题定制：
//   左栏：给定资料+题目输入、可选"我的作答"输入、材料/作答双目标标注视图。
//   右栏：题型胶囊 → 答题方法 → 批改（采分点命中清单，仅在提交作答时）→ 参考答案全文+思路。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { ClipboardCheck, ScrollText } from 'lucide-react'
import type { AIProvider, AnalysisResult, GradingPoint } from '../types/analysis'
import { useAnalysis } from '../hooks/useAnalysis'
import QuestionInput from './QuestionInput'
import MaterialAnnotation from './MaterialAnnotation'
import { ResultPills, TechniquePanel } from './AnalysisResult'

// RestorePayload 是历史恢复时上层注入的一次性数据：nonce 变化即应用一次，
// 让申论记录能把自己的材料/我的作答/分析结果灌回本页独立的状态。
export interface RestorePayload {
  material: string
  userAnswer?: string
  result: AnalysisResult | null
  nonce: number
}

interface Props {
  subjectId: string
  hasSubject: boolean
  providers: AIProvider[]
  activeProviderId: string
  activeModel: string
  onProviderChange: (id: string) => void
  onModelChange: (model: string) => void
  onOpenSettings: () => void
  // 每次分析完成后通知上层刷新历史列表。
  onAnalyzed: () => void
  // 历史恢复：nonce 递增时把 material/answer/result 灌回本页。
  restore?: RestorePayload | null
  // 向上传递本页的加载态，供 TopBar 进度条使用。
  onRunningChange?: (running: boolean) => void
}

export default function ShenlunPage({
  subjectId,
  hasSubject,
  providers,
  activeProviderId,
  activeModel,
  onProviderChange,
  onModelChange,
  onOpenSettings,
  onAnalyzed,
  restore,
  onRunningChange,
}: Props) {
  const [material, setMaterial] = useState('')
  const [answer, setAnswer] = useState('')
  const [error, setError] = useState('')
  const { result, stage, error: analysisError, run, cancel, setResult, partial } = useAnalysis()
  const loading = stage === 'running'

  // 历史恢复：restore.nonce 递增即把 material/answer/result 灌回本页一次。
  const appliedRef = useRef(0)
  useEffect(() => {
    if (!restore || restore.nonce === 0 || restore.nonce === appliedRef.current) return
    appliedRef.current = restore.nonce
    setMaterial(restore.material)
    setAnswer(restore.userAnswer ?? '')
    setError('')
    setResult(restore.result)
  }, [restore, setResult])

  // 加载态上抛，供 TopBar 进度条统一显示；离开本页时复位，避免进度条卡在加载中。
  const reportRunning = useCallback(
    (r: boolean) => onRunningChange?.(r),
    [onRunningChange],
  )
  useEffect(() => {
    reportRunning(loading)
  }, [loading, reportRunning])
  // 组件卸载（切走视图）时复位，避免进度条卡在加载态。
  useEffect(() => () => onRunningChange?.(false), [onRunningChange])

  const handleAnalyze = async () => {
    const trimmed = material.trim()
    if (!hasSubject) {
      setError('申论讲义尚未加载完成，请稍后再试')
      return
    }
    if (trimmed.length < 6) {
      setError('请粘贴给定资料与作答要求')
      return
    }
    setError('')
    await run(subjectId, trimmed, answer.trim() || undefined)
    onAnalyzed()
  }

  const pageError = error || analysisError

  // 材料侧标注 = highlights；作答侧标注由 MaterialAnnotation 依 grading 生成。
  const allHighlights = useMemo(() => result?.highlights ?? [], [result])
  const hasResult = Boolean(result) && !loading
  // 批改面板：提交了我的作答，或恢复的记录本身带批改点时都显示。
  const showGrading = answer.trim().length > 0 || (result?.grading?.points?.length ?? 0) > 0

  return (
    <>
      {/* 左栏：输入（资料+作答）+ 材料/作答标注视图 */}
      <section className="flex h-full w-[57%] min-w-0 flex-col gap-3 max-lg:h-auto max-lg:w-full">
        <div className="flex shrink-0 flex-col gap-3">
          <QuestionInput
            question={material}
            onChange={setMaterial}
            onAnalyze={() => void handleAnalyze()}
            onCancel={cancel}
            loading={loading}
            hasSubject={hasSubject}
            error={pageError}
            providers={providers}
            activeProviderId={activeProviderId}
            activeModel={activeModel}
            onProviderChange={onProviderChange}
            onModelChange={onModelChange}
            onOpenSettings={onOpenSettings}
            placeholder={'请粘贴申论题目：给定资料 + 作答要求。\n例如：\n根据"给定资料2"，概括……（不超过200字）'}
            analyzeLabel={answer.trim() ? '分析并批改' : '开始分析'}
          />
          <AnswerInput value={answer} onChange={setAnswer} disabled={loading} />
        </div>
        <div className="min-h-0 flex-1">
          <MaterialAnnotation
            materialText={material}
            answerText={answer}
            highlights={allHighlights}
            grading={result?.grading}
            active={hasResult}
          />
        </div>
      </section>

      {/* 右栏：题型 / 方法 / 批改 / 参考答案 */}
      <section className="flex h-full min-w-0 flex-1 flex-col gap-3 max-lg:h-auto">
        <ResultPills result={result} loading={loading} partial={partial} />
        <TechniquePanel result={result} loading={loading} partial={partial} />
        {showGrading ? (
          <GradingPanel result={result} loading={loading} />
        ) : (
          <ReferencePanel result={result} loading={loading} />
        )}
      </section>
    </>
  )
}


// ---- 我的作答输入（可选）----

function AnswerInput({ value, onChange, disabled }: { value: string; onChange: (v: string) => void; disabled: boolean }) {
  return (
    <div className="shrink-0 overflow-hidden rounded-card bg-surface shadow-card">
      <div className="flex h-9 items-center gap-2 border-b border-hairline px-4">
        <ClipboardCheck className="h-3.5 w-3.5 text-body" aria-hidden="true" />
        <span className="text-xs font-medium text-muted">我的作答（可选，填写后将按采分点批改）</span>
      </div>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        placeholder="把你自己写的答案粘贴在这里；不填则只生成参考答案，不批改。"
        className="min-h-20 w-full resize-none border-0 bg-transparent p-4 text-[15px] leading-7 text-body outline-none placeholder:text-ghost disabled:opacity-50"
        spellCheck={false}
      />
    </div>
  )
}

// ---- 批改面板：采分点命中清单 ----

function GradingPanel({ result, loading }: { result: AnalysisResult | null; loading: boolean }) {
  const grading = result?.grading
  const points = grading?.points ?? []
  return (
    <Panel title="批改 · 采分点" icon={<ClipboardCheck className="h-3.5 w-3.5" aria-hidden="true" />} className="min-h-0 flex-1">
      {loading ? (
        <p className="text-[13px] text-quiet">正在生成批改…</p>
      ) : !result ? (
        <p className="text-[13px] text-quiet">等待分析结果</p>
      ) : points.length === 0 ? (
        <p className="text-[13px] text-quiet">未产出批改。若已填"我的作答"仍无结果，可重试。</p>
      ) : (
        <div className="space-y-2.5">
          {grading?.summary && (
            <p className="rounded-chip bg-fill px-3 py-2 text-[13px] leading-6 text-body">{grading.summary}</p>
          )}
          {points.map((p, i) => (
            <PointRow key={i} point={p} />
          ))}
        </div>
      )}
    </Panel>
  )
}

const STATUS_META: Record<string, { label: string; cls: string }> = {
  hit: { label: '命中', cls: 'bg-success-tint text-success' },
  partial: { label: '部分', cls: 'bg-heat-tint text-heat' },
  miss: { label: '遗漏', cls: 'bg-danger-tint text-danger' },
}

function PointRow({ point }: { point: GradingPoint }) {
  const meta = STATUS_META[point.status] ?? { label: point.status, cls: 'bg-fill text-muted' }
  return (
    <div className="rounded-card border border-hairline p-3">
      <div className="flex items-start gap-2">
        <span className={`mt-0.5 inline-flex h-5 shrink-0 items-center rounded-chip px-1.5 text-[11px] font-medium ${meta.cls}`}>
          {meta.label}
        </span>
        <p className="min-w-0 flex-1 text-[13px] leading-6 text-ink">{point.point}</p>
      </div>
      {point.suggestion && <p className="mt-1.5 pl-[52px] text-[12px] leading-5 text-muted">建议：{point.suggestion}</p>}
    </div>
  )
}

// ---- 参考答案全文 + 思路（未提交作答时的主面板）----

function ReferencePanel({ result, loading }: { result: AnalysisResult | null; loading: boolean }) {
  return (
    <Panel title="参考答案与思路" icon={<ScrollText className="h-3.5 w-3.5" aria-hidden="true" />} className="min-h-0 flex-1">
      {loading ? (
        <p className="text-[13px] text-quiet">正在生成参考答案…</p>
      ) : !result ? (
        <p className="text-[13px] text-quiet">等待分析结果</p>
      ) : (
        <div className="space-y-3">
          <div>
            <h4 className="mb-1 text-[11px] font-medium text-muted">参考答案</h4>
            <p className="text-[15px] leading-7 whitespace-pre-wrap text-body">{result.answer || '（无）'}</p>
          </div>
          {result.annotation && (
            <div className="border-t border-hairline pt-3">
              <h4 className="mb-1 text-[11px] font-medium text-muted">解题思路</h4>
              <p className="text-[14px] leading-7 whitespace-pre-wrap text-body">{result.annotation}</p>
            </div>
          )}
        </div>
      )}
    </Panel>
  )
}

// ---- 通用卡片（沿用右栏视觉：h-10 头部 + 可滚动正文）----

function Panel({
  title,
  icon,
  className = '',
  children,
}: {
  title: string
  icon?: React.ReactNode
  className?: string
  children: React.ReactNode
}) {
  return (
    <section className={`flex min-h-0 flex-col overflow-hidden rounded-card bg-surface shadow-card ${className}`}>
      <header className="flex h-10 shrink-0 items-center gap-2 border-b border-hairline px-4">
        {icon}
        <h3 className="text-xs font-medium text-muted">{title}</h3>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3.5">{children}</div>
    </section>
  )
}
