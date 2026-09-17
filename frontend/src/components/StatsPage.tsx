import { useCallback, useEffect, useMemo, useState } from 'react'
import { RefreshCw } from 'lucide-react'
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
    <div className="mx-auto h-full w-full max-w-[1080px] overflow-y-auto">
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h2 className="text-[17px] font-semibold tracking-[-0.02em] text-ink">统计</h2>
          <p className="mt-0.5 text-xs text-muted">全部历史记录的汇总数据</p>
        </div>
        <button
          onClick={() => void load()}
          disabled={loading}
          className="inline-flex h-8 items-center gap-1.5 rounded-pill bg-fill px-3.5 text-[13px] font-medium text-body transition-[background-color,transform] duration-150 ease-quart hover:bg-fill-strong active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-40"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} aria-hidden="true" />
          刷新
        </button>
      </div>

      {error ? (
        <div className="rounded-thumb bg-danger-tint px-4 py-3 text-sm text-danger">{error}</div>
      ) : loading && !stats ? (
        <div className="panel grid grid-cols-2 xl:grid-cols-5">
          {Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="h-24 animate-pulse bg-fill" />
          ))}
        </div>
      ) : stats ? (
        <div className="space-y-7">
          {/* 汇总指标：一个面板 + 发丝线分栏 */}
          <div className="panel metric-grid">
            <Metric label="累计题目" value={stats.total_questions.toLocaleString()} />
            <Metric label="总 tokens" value={stats.total_tokens.toLocaleString()} />
            <Metric
              label="平均 tokens / 题"
              value={stats.total_questions > 0 ? Math.round(stats.avg_tokens).toLocaleString() : '0'}
            />
            <Metric
              label="平均首字响应"
              value={
                stats.avg_first_token_ms > 0 ? `${(stats.avg_first_token_ms / 1000).toFixed(1)}s` : '—'
              }
            />
            <Metric label="累计用时" value={formatDuration(stats.total_elapsed_ms)} />
          </div>

          {/* 每日趋势 */}
          <section>
            <h3 className="mb-2.5 text-[13px] font-medium text-muted">每日趋势（近 14 天）</h3>
            <div className="panel px-4 pt-4 pb-2">
              <div className="flex h-36 items-end gap-1">
                {days.map((d) => (
                  <div
                    key={d.day}
                    className="flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-1.5"
                    title={`${d.day}：${d.questions} 题 · ${d.tokens.toLocaleString()} tokens`}
                  >
                    <div className="flex w-full flex-1 items-end">
                      <div
                        className={`w-full rounded-t-[3px] ${d.questions > 0 ? 'bg-ink' : 'bg-track'}`}
                        style={{
                          height:
                            d.questions > 0
                              ? `${Math.max(6, Math.round((d.questions / maxDaily) * 100))}%`
                              : '3px',
                        }}
                      />
                    </div>
                    <span className="tnum shrink-0 text-[9px] text-quiet">{d.day.slice(5)}</span>
                  </div>
                ))}
              </div>
            </div>
          </section>

          {/* 按模型用量 */}
          <section>
            <h3 className="mb-2.5 text-[13px] font-medium text-muted">按模型用量</h3>
            {stats.by_model.length === 0 ? (
              <p className="rounded-thumb border border-dashed border-hint px-4 py-8 text-center text-xs text-quiet">
                暂无模型用量数据
              </p>
            ) : (
              <div className="panel">
                <div className="flex items-center gap-3 border-b border-hairline px-4 py-2">
                  <span className="w-5 shrink-0" />
                  <span className="flex-1 text-[10px] text-quiet">模型</span>
                  <span className="tnum w-12 shrink-0 text-right text-[10px] text-quiet">题数</span>
                  <span className="tnum hidden w-24 shrink-0 text-right text-[10px] text-quiet sm:block">
                    tokens
                  </span>
                  <span className="tnum hidden w-16 shrink-0 text-right text-[10px] text-quiet lg:block">
                    平均首字
                  </span>
                  <span className="tnum hidden w-16 shrink-0 text-right text-[10px] text-quiet lg:block">
                    用时
                  </span>
                </div>
                <div className="divide-y divide-hairline">
                  {stats.by_model.map((m, i) => (
                    <div key={m.model || i} className="flex items-center gap-3 px-4 py-2.5">
                      <span className="tnum w-5 shrink-0 text-center text-[11px] text-hint">
                        {i + 1}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-[13px] text-ink" title={m.model}>
                          {m.model || '默认模型'}
                        </p>
                        <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-pill bg-track">
                          <div
                            className="h-full rounded-pill bg-ink"
                            style={{
                              width: `${Math.max(2, Math.round((m.tokens / maxModelTokens) * 100))}%`,
                            }}
                          />
                        </div>
                      </div>
                      <span className="tnum w-12 shrink-0 text-right text-[11px] text-quiet">
                        {m.questions} 题
                      </span>
                      <span className="tnum hidden w-24 shrink-0 text-right text-xs text-body sm:block">
                        {m.tokens.toLocaleString()}
                      </span>
                      <span className="tnum hidden w-16 shrink-0 text-right text-[11px] text-quiet lg:block">
                        {m.avg_first_token_ms > 0
                          ? `${(m.avg_first_token_ms / 1000).toFixed(1)}s`
                          : '—'}
                      </span>
                      <span className="tnum hidden w-16 shrink-0 text-right text-[11px] text-quiet lg:block">
                        {formatDuration(m.elapsed_ms)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </section>

          {/* 最近分析 */}
          <section>
            <h3 className="mb-2.5 text-[13px] font-medium text-muted">最近分析</h3>
            {recent.length === 0 ? (
              <p className="rounded-thumb border border-dashed border-hint px-4 py-8 text-center text-xs text-quiet">
                暂无分析记录
              </p>
            ) : (
              <ul className="panel divide-y divide-hairline">
                {recent.map((item) => (
                  <li key={item.id} className="flex items-center gap-4 px-4 py-3">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-[13px] text-ink">{item.question}</p>
                      <p className="mt-0.5 text-[11px] text-quiet">{item.created_at}</p>
                    </div>
                    {item.category && (
                      <span className="hidden shrink-0 rounded-pill bg-fill px-2 py-0.5 text-[11px] text-muted sm:inline">
                        {item.category}
                      </span>
                    )}
                    <span className="tnum w-24 shrink-0 text-right text-xs text-muted">
                      {item.tokens.toLocaleString()} tokens
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      ) : null}
    </div>
  )
}

// Metric 是汇总面板里的一格：标签 + 数字，没有图标和装饰。
function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="px-4 py-4">
      <p className="text-[11px] text-quiet">{label}</p>
      <p className="tnum mt-2 text-[26px] font-semibold tracking-[-0.02em] text-ink">{value}</p>
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
