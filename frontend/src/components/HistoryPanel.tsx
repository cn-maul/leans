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
      <div className="animate-pulse space-y-2 p-5">
        <div className="h-16 rounded-lg bg-zinc-100 dark:bg-zinc-800" />
        <div className="h-16 rounded-lg bg-zinc-100 dark:bg-zinc-800" />
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-14 text-center">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-dashed border-zinc-300 text-zinc-300 dark:border-zinc-700 dark:text-zinc-600">
          <History className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="mt-3 text-sm text-zinc-500 dark:text-zinc-400">暂无历史记录</p>
      </div>
    )
  }

  return (
    <ul className="space-y-2 p-4">
      {items.map((item, i) => (
        <li key={item.id} className="animate-fade-up" style={{ animationDelay: `${i * 30}ms` }}>
          <button
            onClick={() => onSelect(item)}
            title="点击回填该题目与分析结果"
            className="group w-full rounded-lg border border-zinc-200 bg-white px-4 py-3 text-left transition-colors hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-zinc-700 dark:hover:bg-zinc-800/60"
          >
            <div className="flex items-center justify-between gap-3">
              <span className="text-[11px] text-zinc-400 dark:text-zinc-500">{item.created_at}</span>
              <span className="flex min-w-0 items-center gap-2">
                {item.model && (
                  <span
                    className="max-w-[110px] truncate text-[11px] text-zinc-400 dark:text-zinc-500"
                    title={`模型 ${item.model}`}
                  >
                    {item.model}
                  </span>
                )}
                <span className="tnum text-[11px] text-zinc-400 dark:text-zinc-500">
                  {item.tokens.toLocaleString()} tokens
                </span>
                <ChevronRight
                  className="h-4 w-4 shrink-0 text-zinc-300 transition-transform group-hover:translate-x-0.5 dark:text-zinc-600"
                  aria-hidden="true"
                />
              </span>
            </div>
            <div className="mt-1.5 line-clamp-2 text-sm leading-6 text-zinc-700 dark:text-zinc-200">
              {item.question}
            </div>
            {item.category && (
              <span className="mt-2 inline-flex items-center rounded-full border border-zinc-200 px-2 py-0.5 text-[11px] text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
                {item.category}
              </span>
            )}
          </button>
        </li>
      ))}
    </ul>
  )
}
