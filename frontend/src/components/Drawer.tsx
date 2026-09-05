import type { ReactNode } from 'react'
import { X } from 'lucide-react'

interface Props {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
}

export default function Drawer({ open, title, onClose, children, footer }: Props) {
  return (
    <div
      className={`fixed inset-0 z-50 transition-opacity duration-200 ${
        open ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0'
      }`}
      aria-hidden={!open}
    >
      <div className="absolute inset-0 bg-zinc-950/40 backdrop-blur-[2px]" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={`absolute top-0 right-0 flex h-full w-full max-w-md flex-col border-l border-zinc-200 bg-white shadow-2xl transition-transform duration-300 dark:border-zinc-800 dark:bg-zinc-900 ${
          open ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-zinc-100 px-5 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{title}</h3>
          <button
            onClick={onClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
            aria-label="关闭"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
        {footer && (
          <div className="shrink-0 border-t border-zinc-100 px-5 py-3 dark:border-zinc-800">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}
