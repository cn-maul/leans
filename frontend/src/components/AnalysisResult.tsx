import { useEffect, useRef, useState } from 'react'
import type { AnalysisResult, PartialAnalysis, Rule } from '../types/analysis'

interface PanelProps {
  result: AnalysisResult | null
  loading?: boolean
  // streaming 为 true 时用 partial 渐进渲染。
  partial?: PartialAnalysis | null
  firstTokenMS?: number
  model?: string
}

function normalizeRules(result: AnalysisResult): Rule[] {
  if (result.technique_judgment?.rules?.length) return result.technique_judgment.rules
  if (result.rules && result.rules.length > 0) return result.rules
  if (result.applicable && result.applicable.length > 0) {
    return result.applicable.map((a) => ({
      name: a.rule_name,
      section: a.section,
      application: a.usage,
      usage: a.usage,
      example: a.example,
    }))
  }
  return []
}

// 实时用时计时器（running 期间每 100ms 刷新）。
function useElapsed(running: boolean): string {
  const [startedAt] = useState(() => Date.now())
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!running) return
    const t = setInterval(() => setNow(Date.now()), 100)
    return () => clearInterval(t)
  }, [running])
  return ((now - startedAt) / 1000).toFixed(1)
}

// ---- 顶部三胶囊：题型 / 技巧名称 / 答案 ----

export function ResultPills({ result, loading, partial }: PanelProps) {
  if (loading && !partial) {
    return (
      <div className="flex shrink-0 items-center gap-2" aria-hidden="true">
        <PillSkeleton className="w-36" />
        <PillSkeleton className="w-44" />
        <PillSkeleton className="w-24" />
      </div>
    )
  }

  if (loading && partial) {
    // 流式中：胶囊随 AI 输出逐步填充。
    const typeText = [partial.category, partial.subCategory].filter(Boolean).join(' · ')
    const techniqueText = partial.rules.map((r) => r.name).join('、')
    return (
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        <Pill label="题型" value={typeText || '…'} delay={0} live />
        <Pill label="技巧" value={techniqueText || '…'} delay={60} live />
        <Pill label="答案" value={partial.answer || '…'} solid delay={120} live />
      </div>
    )
  }

  const tj = result?.type_judgment
  const category = tj?.category || result?.category || ''
  const subCategory = tj?.sub_category || result?.sub_category || ''
  const typeText = [category, subCategory].filter(Boolean).join(' · ')

  const rules = result ? normalizeRules(result) : []
  const techniqueText = rules.map((r) => r.name).join('、')
  // 未分析时显示占位符，而不是"未匹配技巧"。
  const techniqueValue = result ? techniqueText || '未匹配技巧' : '—'

  const answer = result?.answer || ''

  return (
    <div className="flex shrink-0 flex-wrap items-center gap-2">
      <Pill label="题型" value={typeText || '—'} title={typeText} delay={0} />
      <Pill
        label="技巧"
        value={techniqueValue}
        title={rules.map((r) => `${r.name}（${r.section}）`).join('、')}
        delay={60}
      />
      <Pill label="答案" value={answer || '—'} solid delay={120} />
    </div>
  )
}

function PillSkeleton({ className = '' }: { className?: string }) {
  return (
    <div className={`h-9 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800 ${className}`} />
  )
}

function Pill({
  label,
  value,
  title,
  solid,
  delay,
  live,
}: {
  label: string
  value: string
  title?: string
  solid?: boolean
  delay: number
  live?: boolean
}) {
  const isAnswer = label === '答案'
  return (
    <div
      className={`inline-flex h-9 min-w-0 max-w-full animate-fade-up items-center gap-2 rounded-full border pl-3.5 pr-4 transition-colors ${
        solid
          ? 'border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900'
          : 'border-zinc-200 bg-white text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100'
      }`}
      style={{ animationDelay: `${delay}ms` }}
      title={title || value}
    >
      <span
        className={`shrink-0 text-[11px] tracking-wide ${
          solid ? 'text-white/60 dark:text-zinc-900/60' : 'text-zinc-400 dark:text-zinc-500'
        }`}
      >
        {label}
      </span>
      <span
        className={`min-w-0 truncate ${
          isAnswer ? 'text-base font-semibold' : 'text-[13px] font-medium'
        } ${live && value === '…' ? 'animate-pulse-dot' : ''}`}
      >
        {value}
      </span>
    </div>
  )
}

// ---- 通用卡片框 ----

function Card({
  title,
  aside,
  className = '',
  bodyRef,
  children,
}: {
  title: string
  aside?: React.ReactNode
  className?: string
  bodyRef?: React.RefObject<HTMLDivElement | null>
  children: React.ReactNode
}) {
  return (
    <section
      className={`flex min-h-0 animate-fade-up flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900 ${className}`}
    >
      <header className="flex h-10 shrink-0 items-center justify-between gap-3 border-b border-zinc-100 px-4 dark:border-zinc-800">
        <h3 className="text-xs font-medium text-zinc-500 dark:text-zinc-400">{title}</h3>
        {aside}
      </header>
      <div ref={bodyRef} className="min-h-0 flex-1 overflow-y-auto px-4 py-3.5">
        {children}
      </div>
    </section>
  )
}

function SkeletonLines({ count = 3 }: { count?: number }) {
  return (
    <div className="animate-fade-in space-y-2.5" aria-label="加载中">
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className="h-3 animate-pulse rounded bg-zinc-100 dark:bg-zinc-800"
          style={{ width: i === count - 1 ? '55%' : `${88 - i * 12}%` }}
        />
      ))}
    </div>
  )
}

function EmptyHint({ text, hint }: { text: string; hint?: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-1 py-4 text-center">
      <p className="text-xs text-zinc-500 dark:text-zinc-400">{text}</p>
      {hint && <p className="text-[11px] text-zinc-400 dark:text-zinc-500">{hint}</p>}
    </div>
  )
}

// ---- 技巧应用：技巧在这道题里怎么用（不再列举依据） ----

export function TechniquePanel({ result, loading, partial }: PanelProps) {
  const streaming = loading && !!partial && partial.rules.length > 0
  return (
    <Card title="技巧应用" className="max-h-[42%] shrink-0">
      {streaming ? (
        <RulesBody rules={partial.rules} streaming />
      ) : loading ? (
        <SkeletonLines count={2} />
      ) : !result ? (
        <EmptyHint text="等待分析结果" hint="命中的讲义技巧及其用法会显示在这里" />
      ) : (
        <TechniqueBody result={result} />
      )}
    </Card>
  )
}

function RulesBody({
  rules,
  streaming,
}: {
  rules: Array<{ name: string; section: string; usage: string }>
  streaming?: boolean
}) {
  return (
    <div className="space-y-4">
      {rules.map((r, i) => (
        <div key={i} className="animate-fade-up" style={{ animationDelay: `${i * 60}ms` }}>
          <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
            <h4 className="text-[13px] font-semibold text-zinc-900 dark:text-zinc-100">{r.name}</h4>
            {r.section && (
              <span className="text-[11px] text-zinc-400 dark:text-zinc-500">{r.section}</span>
            )}
          </div>
          {r.usage ? (
            <p className="mt-1 text-[13px] leading-6 text-zinc-600 dark:text-zinc-300">
              {r.usage}
              {streaming && i === rules.length - 1 && <Caret />}
            </p>
          ) : (
            <p className="mt-1 text-[13px] text-zinc-400 dark:text-zinc-500">
              正在生成用法…
              {streaming && i === rules.length - 1 && <Caret />}
            </p>
          )}
        </div>
      ))}
    </div>
  )
}

function TechniqueBody({ result }: { result: AnalysisResult }) {
  const rules = normalizeRules(result)
  if (rules.length === 0) {
    return <EmptyHint text="本题未命中讲义技巧" hint="讲义中没有可直接套用的规则" />
  }
  return (
    <div className="space-y-4">
      {rules.map((r, i) => {
        const application = r.application || r.usage || ''
        return (
          <div key={i} className="animate-fade-up" style={{ animationDelay: `${i * 60}ms` }}>
            <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
              <h4 className="text-[13px] font-semibold text-zinc-900 dark:text-zinc-100">
                {r.name}
              </h4>
              {r.section && (
                <span className="text-[11px] text-zinc-400 dark:text-zinc-500">{r.section}</span>
              )}
            </div>
            {application ? (
              <p className="mt-1 text-[13px] leading-6 text-zinc-600 dark:text-zinc-300">
                {application}
              </p>
            ) : (
              <p className="mt-1 text-[13px] text-zinc-400 dark:text-zinc-500">暂无用法说明</p>
            )}
          </div>
        )
      })}
    </div>
  )
}

// ---- 注释和思路：技巧应用之外的剩余内容 ----

export function AnnotationPanel({
  result,
  loading,
  partial,
  firstTokenMS = 0,
  model,
}: PanelProps) {
  const meta = result?.meta
  const hasMeta = !!meta && (meta.elapsed_ms > 0 || meta.total_tokens > 0)
  const streamingText = loading ? (partial?.annotation || '') : ''
  const elapsed = useElapsed(Boolean(loading))
  const bodyRef = useRef<HTMLDivElement | null>(null)

  // 流式输出时自动滚动到底部。
  useEffect(() => {
    const el = bodyRef.current
    if (loading && el) el.scrollTop = el.scrollHeight
  }, [streamingText, loading])

  // 完成后的 meta 信息条：模型 · 首字 · 总用时 · tokens。
  const metaAside = hasMeta ? (
    <div className="tnum flex shrink-0 items-center gap-2 overflow-hidden text-[11px] text-zinc-400 dark:text-zinc-500">
      {meta.model && (
        <span className="truncate" title={`模型 ${meta.model}`}>
          {meta.model}
        </span>
      )}
      {meta.model && (meta.first_token_ms || meta.elapsed_ms > 0) && <span aria-hidden="true">·</span>}
      {!!meta.first_token_ms && meta.first_token_ms > 0 && (
        <span title="首字响应时间">首字 {(meta.first_token_ms / 1000).toFixed(1)}s</span>
      )}
      {meta.elapsed_ms > 0 && (
        <>
          {!!meta.first_token_ms && <span aria-hidden="true">·</span>}
          <span title="处理耗时">{(meta.elapsed_ms / 1000).toFixed(1)}s</span>
        </>
      )}
      {meta.total_tokens > 0 && (
        <>
          <span aria-hidden="true">·</span>
          <span title="token 用量">{meta.total_tokens.toLocaleString()} tokens</span>
        </>
      )}
    </div>
  ) : undefined

  // 流式中的 header：模型 · 首字 · 实时用时。
  const liveAside = loading ? (
    <div className="tnum flex shrink-0 items-center gap-2 overflow-hidden text-[11px] text-zinc-400 dark:text-zinc-500">
      {model && (
        <span className="truncate" title={`模型 ${model}`}>
          {model}
        </span>
      )}
      {model && <span aria-hidden="true">·</span>}
      {firstTokenMS > 0 && (
        <>
          <span title="首字响应时间">首字 {(firstTokenMS / 1000).toFixed(1)}s</span>
          <span aria-hidden="true">·</span>
        </>
      )}
      <span>{elapsed}s</span>
    </div>
  ) : undefined

  return (
    <Card
      title="注释和思路"
      className="min-h-0 flex-1"
      aside={loading ? liveAside : metaAside}
      bodyRef={bodyRef}
    >
      {loading ? (
        streamingText ? (
          <p className="animate-fade-in text-sm leading-7 whitespace-pre-wrap text-zinc-700 dark:text-zinc-300">
            {streamingText}
            <Caret />
          </p>
        ) : (
          <AnalyzingState />
        )
      ) : !result ? (
        <EmptyHint text="等待分析结果" hint="完成分析后在这里查看解题思路" />
      ) : result.annotation ? (
        <p className="animate-fade-in text-sm leading-7 whitespace-pre-wrap text-zinc-700 dark:text-zinc-300">
          {result.annotation}
        </p>
      ) : (
        <p className="text-sm text-zinc-400 dark:text-zinc-500">暂无注释</p>
      )}
    </Card>
  )
}

// 打字机光标。
function Caret() {
  return (
    <span
      className="ml-0.5 inline-block h-4 w-[2px] translate-y-[3px] animate-pulse-dot bg-zinc-800 dark:bg-zinc-200"
      aria-hidden="true"
    />
  )
}

const STAGES = ['正在检索讲义相关章节…', '正在调用 AI 分析题目…', '正在等待模型输出…']

// 首个 token 到达前的等待态：实时用时 + 阶段提示。
function AnalyzingState() {
  const [stage, setStage] = useState(0)
  const elapsed = useElapsed(true)

  useEffect(() => {
    const stageTimer = setInterval(() => {
      setStage((s) => Math.min(s + 1, STAGES.length - 1))
    }, 4000)
    return () => clearInterval(stageTimer)
  }, [])

  return (
    <div className="flex h-full flex-col items-center justify-center gap-3">
      <p className="tnum text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
        {elapsed}s
      </p>
      <p className="text-xs text-zinc-500 dark:text-zinc-400">{STAGES[stage]}</p>
      <div className="flex gap-1.5">
        {STAGES.map((_, i) => (
          <span
            key={i}
            className={`h-1.5 w-1.5 rounded-full transition-colors duration-300 ${
              i === stage
                ? 'animate-pulse-dot bg-zinc-900 dark:bg-zinc-100'
                : i < stage
                  ? 'bg-zinc-400 dark:bg-zinc-500'
                  : 'bg-zinc-200 dark:bg-zinc-700'
            }`}
          />
        ))}
      </div>
    </div>
  )
}
