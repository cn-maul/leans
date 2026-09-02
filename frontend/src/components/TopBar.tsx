import type { Subject, Theme } from '../types/analysis'

interface Props {
  subjects: Subject[]
  selectedSubject: string
  onSelectSubject: (id: string) => void
  onOpenLecture: () => void
  onOpenHistory: () => void
  onOpenSettings: () => void
  theme: Theme
  onToggleTheme: () => void
}

// TopBar 是顶栏导航：品牌 + 科目下拉 + 工具入口（讲义/历史/设置/主题）。
export default function TopBar({
  subjects,
  selectedSubject,
  onSelectSubject,
  onOpenLecture,
  onOpenHistory,
  onOpenSettings,
  theme,
  onToggleTheme,
}: Props) {
  return (
    <header className="sticky top-0 z-40 bg-white/90 dark:bg-gray-900/90 backdrop-blur border-b border-gray-200 dark:border-gray-800">
      <div className="max-w-[1440px] mx-auto px-6 py-3 flex items-center gap-4">
        {/* 品牌 */}
        <div className="flex items-center gap-2.5 mr-2 shrink-0">
          <span className="text-2xl leading-none">📚</span>
          <div>
            <h1 className="font-bold text-gray-900 dark:text-gray-100 leading-tight">
              公考题目分析
            </h1>
            <p className="text-[11px] text-gray-500 dark:text-gray-400 leading-tight">
              基于讲义的智能解析
            </p>
          </div>
        </div>

        {/* 科目下拉 */}
        <div className="min-w-0 flex-1 max-w-xs">
          <label className="sr-only" htmlFor="subject-select">
            选择科目
          </label>
          <select
            id="subject-select"
            value={selectedSubject}
            onChange={(e) => onSelectSubject(e.target.value)}
            className="w-full px-3 py-2 text-sm bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-800 dark:text-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="">选择科目…</option>
            {subjects.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex-1" />

        {/* 工具按钮 */}
        <div className="flex items-center gap-1.5">
          <ToolButton onClick={onOpenLecture} label="讲义" icon="📖" />
          <ToolButton onClick={onOpenHistory} label="历史" icon="🕘" />
          <ToolButton onClick={onOpenSettings} label="设置" icon="⚙" />
          <button
            onClick={onToggleTheme}
            className="px-2.5 py-2 text-sm rounded-lg bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
            title={theme === 'light' ? '切换到暗色模式' : '切换到亮色模式'}
          >
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
        </div>
      </div>
    </header>
  )
}

function ToolButton({
  onClick,
  label,
  icon,
}: {
  onClick: () => void
  label: string
  icon: string
}) {
  return (
    <button
      onClick={onClick}
      className="px-2.5 py-2 text-sm rounded-lg text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors flex items-center gap-1"
    >
      <span>{icon}</span>
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}
