import { BarChart3, History, Moon, PenLine, Settings, Sun } from 'lucide-react'
import type { Theme } from '../types/analysis'

export type View = 'analyze' | 'stats'

interface Props {
  lectureName: string
  view: View
  onViewChange: (v: View) => void
  onOpenHistory: () => void
  onOpenSettings: () => void
  theme: Theme
  onToggleTheme: () => void
  running: boolean
}

export default function TopBar({
  lectureName,
  view,
  onViewChange,
  onOpenHistory,
  onOpenSettings,
  theme,
  onToggleTheme,
  running,
}: Props) {
  return (
    <header className="relative shrink-0 border-b border-zinc-200 bg-white/85 backdrop-blur-xl dark:border-zinc-800 dark:bg-zinc-950/85">
      <div className="mx-auto flex h-14 w-full max-w-[1680px] items-center gap-3 px-5">
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-zinc-900 text-[13px] font-bold text-white dark:bg-zinc-100 dark:text-zinc-900">
            析
          </div>
          <h1 className="truncate text-[15px] font-semibold tracking-tight text-zinc-900 dark:text-white">
            公考题目分析
          </h1>
          <span
            className="hidden items-center gap-1 rounded-full border border-zinc-200 px-2.5 py-0.5 text-[11px] text-zinc-500 md:inline-flex dark:border-zinc-800 dark:text-zinc-400"
            title="当前使用的讲义"
          >
            {lectureName}
          </span>
        </div>

        <div className="ml-auto flex items-center gap-1.5">
          <nav className="flex items-center rounded-lg bg-zinc-100 p-0.5 dark:bg-zinc-800/80">
            <ViewTab
              active={view === 'analyze'}
              onClick={() => onViewChange('analyze')}
              icon={<PenLine className="h-3.5 w-3.5" aria-hidden="true" />}
              label="分析"
            />
            <ViewTab
              active={view === 'stats'}
              onClick={() => onViewChange('stats')}
              icon={<BarChart3 className="h-3.5 w-3.5" aria-hidden="true" />}
              label="统计"
            />
          </nav>

          <span className="mx-1 h-5 w-px bg-zinc-200 dark:bg-zinc-800" aria-hidden="true" />

          <IconButton onClick={onOpenHistory} label="历史">
            <History className="h-4 w-4" aria-hidden="true" />
          </IconButton>
          <IconButton onClick={onOpenSettings} label="设置">
            <Settings className="h-4 w-4" aria-hidden="true" />
          </IconButton>
          <IconButton
            onClick={onToggleTheme}
            label={theme === 'light' ? '切换到暗色模式' : '切换到亮色模式'}
          >
            {theme === 'light' ? (
              <Moon className="h-4 w-4" aria-hidden="true" />
            ) : (
              <Sun className="h-4 w-4" aria-hidden="true" />
            )}
          </IconButton>
        </div>
      </div>

      {/* 分析进行中：顶栏底部的不确定进度条 */}
      {running && (
        <div className="absolute inset-x-0 bottom-0 overflow-hidden" aria-hidden="true">
          <div className="h-0.5 w-1/4 rounded-full bg-zinc-900 animate-progress-slide dark:bg-zinc-100" />
        </div>
      )}
    </header>
  )
}

function ViewTab({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
}) {
  return (
    <button
      onClick={onClick}
      className={`inline-flex h-7 items-center gap-1.5 rounded-md px-3 text-[13px] font-medium transition-all ${
        active
          ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-600 dark:text-white'
          : 'text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200'
      }`}
      aria-pressed={active}
    >
      {icon}
      {label}
    </button>
  )
}

function IconButton({
  onClick,
  label,
  children,
}: {
  onClick: () => void
  label: string
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      title={label}
      aria-label={label}
      className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
    >
      {children}
    </button>
  )
}
