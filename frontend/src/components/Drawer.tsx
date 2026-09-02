import type { ReactNode } from 'react'

interface Props {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
}

// Drawer 是一个从右侧滑出的抽屉面板，用于承载讲义浏览与历史记录。
export default function Drawer({ open, title, onClose, children, footer }: Props) {
  return (
    <div
      className={`fixed inset-0 z-50 transition-opacity ${open ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0'}`}
      aria-hidden={!open}
    >
      {/* 遮罩 */}
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      {/* 面板 */}
      <div
        className={`absolute top-0 right-0 h-full w-full max-w-md bg-white dark:bg-gray-900 shadow-2xl transition-transform duration-300 ${
          open ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-gray-800">
          <h3 className="font-semibold text-gray-800 dark:text-gray-100">{title}</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 text-xl leading-none transition-colors"
            aria-label="关闭"
          >
            &times;
          </button>
        </div>
        <div className="h-[calc(100%-57px)] overflow-y-auto">{children}</div>
        {footer && (
          <div className="px-4 py-3 border-t border-gray-100 dark:border-gray-800">{footer}</div>
        )}
      </div>
    </div>
  )
}
