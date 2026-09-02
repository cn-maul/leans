import type { HistoryItem } from '../types/analysis'

interface Props {
  items: HistoryItem[]
  onSelect: (item: HistoryItem) => void
  loading?: boolean
}

// HistoryPanel 在抽屉内展示历史列表，点击条目回填题目与分析结果。
export default function HistoryPanel({ items, onSelect, loading }: Props) {
  if (loading) {
    return (
      <div className="space-y-2 animate-pulse">
        <div className="h-12 bg-gray-100 dark:bg-gray-800 rounded-lg" />
        <div className="h-12 bg-gray-100 dark:bg-gray-800 rounded-lg" />
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="text-sm text-gray-400 dark:text-gray-500 text-center py-6">
        暂无历史记录
      </div>
    )
  }

  return (
    <ul className="space-y-2">
      {items.map((item) => (
        <li key={item.id}>
          <button
            onClick={() => onSelect(item)}
            title="点击回填该题目与分析结果"
            className="w-full text-left p-3 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 hover:border-primary-300 dark:hover:border-primary-700 transition-colors"
          >
            <div className="text-[11px] text-gray-400 dark:text-gray-500">{item.created_at}</div>
            <div className="text-sm text-gray-700 dark:text-gray-200 line-clamp-2 mt-1">
              {item.question}
            </div>
            {item.category && (
              <span className="inline-block mt-1.5 px-2 py-0.5 bg-primary-50 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300 text-xs rounded-full">
                {item.category}
              </span>
            )}
          </button>
        </li>
      ))}
    </ul>
  )
}
