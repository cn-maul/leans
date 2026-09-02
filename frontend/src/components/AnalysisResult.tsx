import { useEffect, useState } from 'react'
import type { AnalysisResult, Highlight, HighlightColor, Rule } from '../types/analysis'
import { MODULE_COLOR } from '../types/analysis'

interface Props {
  result: AnalysisResult | null
  loading?: boolean
}

// 分析阶段提示文案：真实后端为单次请求，这里用时间推进模拟阶段反馈。
const STAGES = [
  '正在检索讲义相关章节…',
  '正在调用 AI 分析题目…',
  '正在解析结果与匹配技巧…',
]

export default function AnalysisResultPanel({ result, loading }: Props) {
  if (loading) return <AnalysisLoading />
  if (!result) return <EmptyState />
  return <ResultView result={result} />
}

function ResultView({ result }: { result: AnalysisResult }) {
  const basis = result.basis ?? []
  const rules = normalizeRules(result)
  const meta = result.meta
  const hasMeta = !!meta && (meta.elapsed_ms > 0 || meta.total_tokens > 0)

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 shadow-sm overflow-hidden">
      {/* 顶部：分类 + 耗时/token */}
      <div className="px-5 py-4 border-b border-gray-100 dark:border-gray-800">
        <div className="flex items-center gap-2 flex-wrap mb-1.5">
          {result.category && (
            <span className="px-3 py-1 bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 text-sm font-medium rounded-full">
              {result.category}
            </span>
          )}
          {result.sub_category && (
            <span className="px-3 py-1 bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300 text-sm font-medium rounded-full">
              {result.sub_category}
            </span>
          )}
          {result.answer && (
            <span className="px-3 py-1 bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300 text-sm font-semibold rounded-full">
              答案 {result.answer}
            </span>
          )}
        </div>
        {hasMeta && (
          <div className="text-[11px] text-gray-400 dark:text-gray-500 flex items-center gap-3">
            <span title="处理耗时">⏱ {(meta.elapsed_ms / 1000).toFixed(1)}s</span>
            {meta.total_tokens > 0 && (
              <span title="token 用量">🔤 {meta.total_tokens} tokens</span>
            )}
          </div>
        )}
      </div>

      <div className="p-4 space-y-4">
        {/* 模块一：题目匹配（蓝） */}
        <SectionCard title="题目匹配" desc="根据哪些词句判断题型" accent="blue">
          {basis.length === 0 ? (
            <p className="text-sm text-gray-400 dark:text-gray-500">
              判断依据缺失（旧版数据）
            </p>
          ) : (
            <ul className="space-y-2">
              {basis.map((h, i) => (
                <BasisItem key={i} h={h} />
              ))}
            </ul>
          )}
        </SectionCard>

        {/* 模块二：适用规则（绿） */}
        <SectionCard
          title="适用规则"
          desc="文段与选项中命中的技巧"
          accent="green"
        >
          {rules.length === 0 ? (
            <p className="text-sm text-gray-400 dark:text-gray-500">
              未匹配到适用规则
            </p>
          ) : (
            <div className="space-y-3">
              {rules.map((r, i) => (
                <RuleCard key={i} rule={r} />
              ))}
            </div>
          )}
        </SectionCard>

        {/* 模块三：注释和思路（紫） */}
        <SectionCard title="注释和思路" desc="答案解析与解题思路" accent="purple">
          {result.annotation ? (
            <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">
              {result.annotation}
            </p>
          ) : (
            <p className="text-sm text-gray-400 dark:text-gray-500">暂无注释</p>
          )}
        </SectionCard>
      </div>
    </div>
  )
}

// ---- 子模块 ----

function SectionCard({
  title,
  desc,
  accent,
  children,
}: {
  title: string
  desc: string
  accent: HighlightColor
  children: React.ReactNode
}) {
  const border = {
    blue: 'border-blue-200 dark:border-blue-800',
    green: 'border-green-200 dark:border-green-800',
    purple: 'border-purple-200 dark:border-purple-800',
    red: 'border-red-200 dark:border-red-800',
    yellow: 'border-yellow-200 dark:border-yellow-800',
  }[accent]
  const titleCls = {
    blue: 'text-blue-700 dark:text-blue-300',
    green: 'text-green-700 dark:text-green-300',
    purple: 'text-purple-700 dark:text-purple-300',
    red: 'text-red-700 dark:text-red-300',
    yellow: 'text-yellow-700 dark:text-yellow-300',
  }[accent]

  return (
    <div className={`rounded-xl border ${border} overflow-hidden`}>
      <div className={`px-3 py-2 flex items-center gap-2 bg-opacity-40 ${BG_SOFT[accent]}`}>
        <span className={`w-2.5 h-2.5 rounded-full ${DOT_CLASS[accent]}`} />
        <div>
          <div className={`text-sm font-semibold ${titleCls}`}>{title}</div>
          <div className="text-[11px] text-gray-400 dark:text-gray-500">{desc}</div>
        </div>
      </div>
      <div className="p-3">{children}</div>
    </div>
  )
}

const BG_SOFT: Record<string, string> = {
  blue: 'bg-blue-50 dark:bg-blue-900/20',
  green: 'bg-green-50 dark:bg-green-900/20',
  purple: 'bg-purple-50 dark:bg-purple-900/20',
  red: 'bg-red-50 dark:bg-red-900/20',
  yellow: 'bg-yellow-50 dark:bg-yellow-900/20',
}

const DOT_CLASS: Record<string, string> = {
  blue: 'bg-blue-500',
  green: 'bg-green-500',
  purple: 'bg-purple-500',
  red: 'bg-red-500',
  yellow: 'bg-yellow-500',
}

function BasisItem({ h }: { h: Highlight }) {
  const color = MODULE_COLOR[h.module || ''] || h.color || 'blue'
  return (
    <li className="flex items-start gap-2 text-sm">
      <span className={`inline-block mt-1 w-2 h-2 rounded-full shrink-0 ${DOT_CLASS[color]}`} />
      <div>
        <span className="text-gray-800 dark:text-gray-100 font-medium">{h.text}</span>
        {h.location && (
          <span className="ml-1.5 text-[11px] text-gray-400 dark:text-gray-500">{h.location}</span>
        )}
        {h.explanation && (
          <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{h.explanation}</div>
        )}
      </div>
    </li>
  )
}

function RuleCard({ rule }: { rule: Rule }) {
  const marks = rule.marks ?? []
  return (
    <div className="border border-green-200 dark:border-green-800 rounded-lg p-3 bg-green-50/40 dark:bg-green-900/10">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="font-medium text-green-800 dark:text-green-300 text-sm">{rule.name}</span>
        {rule.section && (
          <span className="text-xs text-green-600 dark:text-green-400">{rule.section}</span>
        )}
      </div>
      {rule.usage && (
        <div className="text-sm text-gray-700 dark:text-gray-300 mt-1.5 leading-relaxed">
          {rule.usage}
        </div>
      )}
      {rule.example && (
        <div className="text-xs text-gray-500 dark:text-gray-400 mt-1 italic">{rule.example}</div>
      )}
      {marks.length > 0 && (
        <div className="mt-2 pt-2 border-t border-green-100 dark:border-green-900">
          <div className="text-[11px] text-green-600 dark:text-green-400 mb-1">命中的词句</div>
          <ul className="space-y-1">
            {marks.map((m, j) => (
              <li key={j} className="flex items-start gap-1.5 text-xs">
                <span className="inline-block mt-1 w-2 h-2 rounded-full bg-green-500 shrink-0" />
                <div>
                  <span className="text-gray-700 dark:text-gray-200 font-medium">{m.text}</span>
                  {m.location && (
                    <span className="ml-1 text-gray-400 dark:text-gray-500">{m.location}</span>
                  )}
                  {m.explanation && (
                    <div className="text-gray-500 dark:text-gray-400">{m.explanation}</div>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}

// 兼容旧数据：把 applicable 映射为 rules。
function normalizeRules(result: AnalysisResult): Rule[] {
  if (result.rules && result.rules.length > 0) return result.rules
  if (result.applicable && result.applicable.length > 0) {
    return result.applicable.map((a) => ({
      name: a.rule_name,
      section: a.section,
      usage: a.usage,
      example: a.example,
    }))
  }
  return []
}

function EmptyState() {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 shadow-sm p-8 flex flex-col items-center justify-center text-center">
      <div className="text-4xl mb-3">📋</div>
      <div className="text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">
        分析结果将显示在这里
      </div>
      <p className="text-xs text-gray-400 dark:text-gray-500 max-w-[240px] leading-relaxed">
        选择科目并粘贴题目，点击「开始分析」，这里会展示题目匹配、适用规则与注释思路
      </p>
    </div>
  )
}

function AnalysisLoading() {
  const [stage, setStage] = useState(0)

  useEffect(() => {
    const timer = setInterval(() => {
      setStage((s) => Math.min(s + 1, STAGES.length - 1))
    }, 1500)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 shadow-sm p-8 flex flex-col items-center justify-center">
      <div className="relative w-12 h-12 mb-4">
        <div className="absolute inset-0 rounded-full border-4 border-blue-200 dark:border-blue-800" />
        <div className="absolute inset-0 rounded-full border-4 border-transparent border-t-blue-600 animate-spin" />
      </div>
      <div className="text-sm text-gray-600 dark:text-gray-300">{STAGES[stage]}</div>
      <div className="mt-3 flex gap-1.5">
        {STAGES.map((_, i) => (
          <span
            key={i}
            className={`w-1.5 h-1.5 rounded-full transition-colors ${
              i <= stage ? 'bg-blue-500' : 'bg-gray-200 dark:bg-gray-700'
            }`}
          />
        ))}
      </div>
    </div>
  )
}
