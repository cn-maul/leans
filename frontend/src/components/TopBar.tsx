import { BarChart3, History, Moon, PenLine, Settings, ScrollText, Sun } from 'lucide-react'
import type { Theme } from '../types/analysis'

export type View = 'analyze' | 'shenlun' | 'stats'

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
    <header className="glass-nav relative shrink-0 border-b border-hairline">
      <div className="mx-auto flex h-14 w-full max-w-[1680px] items-center gap-3 px-5">
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-chip bg-ink text-[13px] font-bold text-surface">
            析
          </div>
          <h1 className="truncate text-[15px] font-semibold tracking-[-0.01em] text-ink">
            公考题目分析
          </h1>
          <span className="hidden items-center rounded-pill bg-fill px-2.5 py-1 text-[11px] text-muted md:inline-flex">
            {lectureName}
          </span>
        </div>

        <div className="ml-auto flex items-center gap-1">
          <nav className="flex items-center gap-[3px] rounded-pill bg-fill p-[3px]">
            <ViewTab
              active={view === 'analyze'}
              onClick={() => onViewChange('analyze')}
              icon={<PenLine className="h-3.5 w-3.5" aria-hidden="true" />}
              label="言语"
            />
            <ViewTab
              active={view === 'shenlun'}
              onClick={() => onViewChange('shenlun')}
              icon={<ScrollText className="h-3.5 w-3.5" aria-hidden="true" />}
              label="申论"
            />
            <ViewTab
              active={view === 'stats'}
              onClick={() => onViewChange('stats')}
              icon={<BarChart3 className="h-3.5 w-3.5" aria-hidden="true" />}
              label="统计"
            />
          </nav>

          <span className="mx-1.5 h-5 w-px bg-hairline" aria-hidden="true" />

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
          <div className="h-0.5 w-1/4 animate-progress-slide rounded-pill bg-ink" />
        </div>
      )}
    </header>
  )
}

// 分段胶囊：灰轨道 + 单个滑动药丸。选中态用白药丸（轻量切换语汇）。
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
      className={`inline-flex h-7 items-center gap-1.5 rounded-pill px-3 text-[13px] transition-all duration-[250ms] ease-quart active:scale-[0.97] ${
        active
          ? 'bg-surface font-semibold text-ink shadow-[0_1px_3px_rgb(0_0_0/0.12)]'
          : 'font-medium text-muted hover:text-ink'
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
      className="inline-flex h-9 w-9 items-center justify-center rounded-pill text-muted transition-[background-color,color,transform] duration-150 ease-quart hover:bg-fill hover:text-ink active:scale-[0.94] max-sm:h-11 max-sm:w-11"
    >
      {children}
    </button>
  )
}
