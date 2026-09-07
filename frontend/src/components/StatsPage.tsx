import { useCallback, useEffect, useMemo, useState } from 'react'
import { BarChart3, Clock, Coins, Cpu, Gauge, RefreshCw, Target, Zap } from 'lucide-react'
import * as api from '../api/client'
import type { DayStat, HistoryItem, Stats } from '../types/analysis'

// 统计视图：累计题数、总 token、平均 token/题、累计用时 + 最近分析列表。
export default function StatsPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [recent, setRecent] = useState<HistoryItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [s, history] = await Promise.all([api.fetchStats(), api.fetchHistory()])
      setStats(s)
      setRecent(history.slice(0, 8))
    } catch (e) {
      setError(e instanceof Error ? e.message : '统计信息加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  // 近 14 天每日题量（无记录的日期补零对齐）。
  const days = useMemo<DayStat[]>(() => {
    const by: Record<string, DayStat> = {}
    for (const d of stats?.by_day ?? []) by[d.day] = d
    const out: DayStat[] = []
    const now = new Date()
    for (let i = 13; i >= 0; i--) {
      const dt = new Date(now.getFullYear(), now.getMonth(), now.getDate() - i)
      const key = `${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, '0')}-${String(dt.getDate()).padStart(2, '0')}`
      out.push(by[key] ?? { day: key, questions: 0, tokens: 0 })
    }
    return out
  }, [stats])
  const maxDaily = Math.max(1, ...days.map((d) => d.questions))
  const maxModelTokens = Math.max(1, ...(stats?.by_model ?? []).map((m) => m.tokens))

  return (
    <div className="mx-auto h-full w-full max-w-4xl animate-fade-up overflow-y-auto">
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold tracking-tight text-zinc-900 dark:text-white">
            统计
          </h2>
          <p className="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">
            全部历史记录的汇总数据
          </p>
        </div>
        <button
          onClick={() => void load()}
          className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-zinc-200 bg-white px-3 text-[13px] text-zinc-600 transition-colors hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800"
          disabled={loading}
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} aria-hidden="true" />
          刷新
        </button>
      </div>

      {error ? (
        <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-600 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-300">
          {error}
        </div>
      ) : loading && !stats ? (
        <div className="grid grid-cols-2 gap-3 xl:grid-cols-5">
          {Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="h-24 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800/70" />
          ))}
        </div>
      ) : stats ? (
        <>
          <div className="grid grid-cols-2 gap-3 xl:grid-cols-5">
            <StatCard
              icon={<Target className="h-3.5 w-3.5" aria-hidden="true" />}
              label="累计题目"
              value={stats.total_questions.toLocaleString()}
              delay={0}
            />
            <StatCard
              icon={<Coins className="h-3.5 w-3.5" aria-hidden="true" />}
              label="总 Tokens"
              value={stats.total_tokens.toLocaleString()}
              delay={60}
            />
            <StatCard
              icon={<Gauge className="h-3.5 w-3.5" aria-hidden="true" />}
              label="平均 Tokens / 题"
              value={stats.total_questions > 0 ? Math.round(stats.avg_tokens).toLocaleString() : '0'}
              delay={120}
            />
            <StatCard
              icon={<Zap className="h-3.5 w-3.5" aria-hidden="true" />}
              label="平均首字响应"
              value={stats.avg_first_token_ms > 0 ? `${(stats.avg_first_token_ms / 1000).toFixed(1)}s` : '—'}
              delay={180}
            />
            <StatCard
              icon={<Clock className="h-3.5 w-3.5" aria-hidden="true" />}
              label="累计用时"
              value={formatDuration(stats.total_elapsed_ms)}
              delay={240}
            />
          </div>

          {/* 每日趋势 */}
          <div className="mt-6">
            <h3 className="mb-2 flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
              <BarChart3 className="h-3.5 w-3.5" aria-hidden="true" />
              每日趋势（近 14 天）
            </h3>
            <div className="rounded-xl border border-zinc-200 bg-white px-4 pt-4 pb-2 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="flex h-36 items-end gap-1">
                {days.map((d) => (
                  <div
                    key={d.day}
                    className="flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-1.5"
                    title={`${d.day}：${d.questions} 题 · ${d.tokens.toLocaleString()} tokens`}
                  >
                    <div className="flex w-full flex-1 items-end">
                      <div
                        className={`w-full rounded-t-md ${
                          d.questions > 0
                            ? 'bg-zinc-900 dark:bg-zinc-100'
                            : 'bg-zinc-200/70 dark:bg-zinc-800'
                        }`}
                        style={{
                          height:
                            d.questions > 0
                              ? `${Math.max(6, Math.round((d.questions / maxDaily) * 100))}%`
                              : '3px',
                        }}
                      />
                    </div>
                    <span className="shrink-0 text-[9px] tabular-nums text-zinc-400 dark:text-zinc-500">
                      {d.day.slice(5)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* 按模型用量 */}
          <div className="mt-6">
            <h3 className="mb-2 flex items-center gap-1.5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
              <Cpu className="h-3.5 w-3.5" aria-hidden="true" />
              按模型用量
            </h3>
            {stats.by_model.length === 0 ? (
              <p className="rounded-xl border border-dashed border-zinc-200 px-4 py-8 text-center text-xs text-zinc-400 dark:border-zinc-800 dark:text-zinc-500">
                暂无模型用量数据
              </p>
            ) : (
              <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
                <div className="flex items-center gap-3 border-b border-zinc-100 px-4 py-1.5 dark:border-zinc-800">
                  <span className="w-5 shrink-0" />
                  <span className="flex-1 text-[10px] text-zinc-400 dark:text-zinc-500">模型</span>
                  <span className="tnum w-12 shrink-0 text-right text-[10px] text-zinc-400 dark:text-zinc-500">
                    题数
                  </span>
                  <span className="tnum hidden w-24 shrink-0 text-right text-[10px] text-zinc-400 sm:block dark:text-zinc-500">
                    Tokens
                  </span>
                  <span className="tnum hidden w-16 shrink-0 text-right text-[10px] text-zinc-400 lg:block dark:text-zinc-500">
                    平均首字
                  </span>
                  <span className="tnum hidden w-16 shrink-0 text-right text-[10px] text-zinc-400 lg:block dark:text-zinc-500">
                    用时
                  </span>
                </div>
                <div className="divide-y divide-zinc-100 dark:divide-zinc-800">
                  {stats.by_model.map((m, i) => (
                    <div key={m.model || i} className="flex items-center gap-3 px-4 py-2.5">
                      <span className="tnum w-5 shrink-0 text-center text-[11px] text-zinc-400 dark:text-zinc-500">
                        {i + 1}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p
                          className="truncate text-[13px] text-zinc-800 dark:text-zinc-200"
                          title={m.model}
                        >
                          {m.model || '默认模型'}
                        </p>
                        <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                          <div
                            className="h-full rounded-full bg-zinc-900 dark:bg-zinc-100"
                            style={{
                              width: `${Math.max(2, Math.round((m.tokens / maxModelTokens) * 100))}%`,
                            }}
                          />
                        </div>
                      </div>
                      <span className="tnum w-12 shrink-0 text-right text-[11px] text-zinc-400 dark:text-zinc-500">
                        {m.questions} 题
                      </span>
                      <span className="tnum hidden w-24 shrink-0 text-right text-xs text-zinc-600 sm:block dark:text-zinc-300">
                        {m.tokens.toLocaleString()} tokens
                      </span>
                      <span className="tnum hidden w-16 shrink-0 text-right text-[11px] text-zinc-400 lg:block dark:text-zinc-500">
                        {m.avg_first_token_ms > 0 ? `${(m.avg_first_token_ms / 1000).toFixed(1)}s` : '—'}
                      </span>
                      <span className="tnum hidden w-16 shrink-0 text-right text-[11px] text-zinc-400 lg:block dark:text-zinc-500">
                        {formatDuration(m.elapsed_ms)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="mt-6">
            <h3 className="mb-2 text-xs font-medium text-zinc-500 dark:text-zinc-400">最近分析</h3>
            {recent.length === 0 ? (
              <p className="rounded-xl border border-dashed border-zinc-200 px-4 py-8 text-center text-xs text-zinc-400 dark:border-zinc-800 dark:text-zinc-500">
                暂无分析记录
              </p>
            ) : (
              <ul className="divide-y divide-zinc-100 overflow-hidden rounded-xl border border-zinc-200 bg-white dark:divide-zinc-800 dark:border-zinc-800 dark:bg-zinc-900">
                {recent.map((item, i) => (
                  <li
                    key={item.id}
                    className="animate-fade-up flex items-center gap-4 px-4 py-3"
                    style={{ animationDelay: `${i * 40}ms` }}
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-[13px] text-zinc-800 dark:text-zinc-200">
                        {item.question}
                      </p>
                      <p className="mt-0.5 text-[11px] text-zinc-400 dark:text-zinc-500">
                        {item.created_at}
                      </p>
                    </div>
                    {item.category && (
                      <span className="hidden shrink-0 rounded-full border border-zinc-200 px-2 py-0.5 text-[11px] text-zinc-500 sm:inline dark:border-zinc-700 dark:text-zinc-400">
                        {item.category}
                      </span>
                    )}
                    <span className="tnum w-24 shrink-0 text-right text-xs text-zinc-500 dark:text-zinc-400">
                      {item.tokens.toLocaleString()} tokens
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}

function StatCard({
  icon,
  label,
  value,
  delay,
}: {
  icon: React.ReactNode
  label: string
  value: string
  delay: number
}) {
  return (
    <div
      className="animate-fade-up rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
      style={{ animationDelay: `${delay}ms` }}
    >
      <div className="flex items-center gap-1.5 text-[11px] text-zinc-400 dark:text-zinc-500">
        {icon}
        {label}
      </div>
      <p className="tnum mt-2 text-2xl font-semibold tracking-tight text-zinc-900 dark:text-white">
        {value}
      </p>
    </div>
  )
}

// ms → "42s" / "3 分 12 秒" / "1 小时 5 分"
function formatDuration(ms: number): string {
  if (ms <= 0) return '0s'
  const totalSec = Math.round(ms / 1000)
  if (totalSec < 60) return `${totalSec}s`
  const min = Math.floor(totalSec / 60)
  const sec = totalSec % 60
  if (min < 60) return sec > 0 ? `${min} 分 ${sec} 秒` : `${min} 分`
  const hour = Math.floor(min / 60)
  return `${hour} 小时 ${min % 60} 分`
}
