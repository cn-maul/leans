import { useEffect, type ReactNode } from 'react'
import { X } from 'lucide-react'

interface Props {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
}

export default function Drawer({ open, title, onClose, children, footer }: Props) {
  // 抽屉始终挂载：卸载会把退出动画一并带走，也没有可中断的当前值可用。
  // 打开状态交给 data-open，由 CSS transition 负责进出场（同一路径、可随时反向）。
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  return (
    <div
      className={`fixed inset-0 z-50 ${open ? '' : 'pointer-events-none'}`}
      aria-hidden={!open}
      inert={!open}
    >
      <div className="scrim absolute inset-0" data-open={open} onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        data-open={open}
        className="drawer glass-sheet absolute top-0 right-0 flex h-full w-full max-w-md flex-col border-l border-hairline shadow-overlay"
      >
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-hairline px-5">
          <h3 className="text-[15px] font-semibold tracking-[-0.01em] text-ink">{title}</h3>
          <button
            onClick={onClose}
            className="inline-flex h-9 w-9 items-center justify-center rounded-pill text-quiet transition-[background-color,color,transform] duration-150 ease-quart hover:bg-fill hover:text-ink active:scale-[0.94] max-sm:h-11 max-sm:w-11"
            aria-label="关闭"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
        {footer && (
          <div className="shrink-0 border-t border-hairline px-5 py-3">{footer}</div>
        )}
      </div>
    </div>
  )
}
