import { ChevronRight, History } from 'lucide-react'
import type { HistoryItem } from '../types/analysis'

interface Props {
  items: HistoryItem[]
  onSelect: (item: HistoryItem) => void
  loading?: boolean
}

export default function HistoryPanel({ items, onSelect, loading }: Props) {
  if (loading) {
    return (
      <div className="space-y-2 p-5">
        <div className="h-16 animate-pulse rounded-thumb bg-fill" />
        <div className="h-16 animate-pulse rounded-thumb bg-fill" />
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-14 text-center">
        <div className="flex h-11 w-11 items-center justify-center rounded-thumb border border-dashed border-hint text-ghost">
          <History className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="mt-3 text-sm text-muted">暂无历史记录</p>
      </div>
    )
  }

  // 同类兄弟项共用一条连续表面，靠发丝线分隔 —— 不是各自成卡。
  return (
    <ul className="divide-y divide-hairline">
      {items.map((item) => (
        <li key={item.id}>
          <button
            onClick={() => onSelect(item)}
            title="点击回填该题目与分析结果"
            className="group w-full px-5 py-3.5 text-left transition-colors duration-150 ease-quart hover:bg-fill active:bg-fill-strong"
          >
            <div className="flex items-center justify-between gap-3">
              <span className="text-[11px] text-quiet">{item.created_at}</span>
              <span className="flex min-w-0 items-center gap-2">
                {item.model && (
                  <span className="max-w-[110px] truncate text-[11px] text-quiet" title={`模型 ${item.model}`}>
                    {item.model}
                  </span>
                )}
                <span className="tnum text-[11px] text-quiet">
                  {item.tokens.toLocaleString()} tokens
                </span>
                <ChevronRight
                  className="h-4 w-4 shrink-0 text-hint transition-transform duration-200 ease-quart group-hover:translate-x-0.5"
                  aria-hidden="true"
                />
              </span>
            </div>
            <div className="mt-1.5 line-clamp-2 text-[14px] leading-6 text-ink">
              {item.question}
            </div>
            {item.category && (
              <span className="mt-2 inline-flex items-center rounded-pill bg-fill px-2 py-0.5 text-[11px] text-muted">
                {item.category}
              </span>
            )}
          </button>
        </li>
      ))}
    </ul>
  )
}
