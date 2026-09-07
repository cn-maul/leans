import { useLayoutEffect, useRef, useState } from 'react'
import { AlertCircle, ChevronDown, CircleStop, ClipboardPaste, Send } from 'lucide-react'
import type { AIProvider } from '../types/analysis'

interface Props {
  question: string
  onChange: (val: string) => void
  onAnalyze: () => void
  onCancel: () => void
  loading: boolean
  hasSubject: boolean
  error: string
  providers: AIProvider[]
  activeProviderId: string
  activeModel: string
  onProviderChange: (id: string) => void
  onModelChange: (model: string) => void
  onOpenSettings: () => void
}

const selectCls =
  'h-7 max-w-[150px] appearance-none rounded-lg border border-zinc-200 bg-white py-0 pl-2.5 pr-7 text-xs text-zinc-700 transition-colors hover:border-zinc-300 focus:border-zinc-400 focus:outline-none focus:ring-2 focus:ring-zinc-900/10 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-950/60 dark:text-zinc-200 dark:hover:border-zinc-600 dark:focus:ring-zinc-100/10'

// 紧凑输入卡：textarea 固定压缩高度，底栏左侧为供应商/模型二级下拉，
// 错误就地显示，替代页面顶部 banner。分析进行中提交按钮变为「中止」。
export default function QuestionInput({
  question,
  onChange,
  onAnalyze,
  onCancel,
  loading,
  hasSubject,
  error,
  providers,
  activeProviderId,
  activeModel,
  onProviderChange,
  onModelChange,
  onOpenSettings,
}: Props) {
  const disabled = loading || !question.trim() || !hasSubject
  const activeProvider = providers.find((p) => p.id === activeProviderId)

  // 「粘贴」：用剪贴板内容整体替换输入框（清空 + 粘贴），失败时短暂提示。
  const [pasteMsg, setPasteMsg] = useState('')
  const pasteTimer = useRef<number | undefined>(undefined)

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText()
      onChange(text)
    } catch {
      setPasteMsg('无法读取剪贴板，请检查浏览器权限')
      window.clearTimeout(pasteTimer.current)
      pasteTimer.current = window.setTimeout(() => setPasteMsg(''), 3000)
    }
  }

  // 输入框随内容自动伸缩（96–320px），长题目不会挤占下方标注视图。
  // Chrome 会把 placeholder 计入 scrollHeight，空输入时须临时摘掉再测量。
  const taRef = useRef<HTMLTextAreaElement>(null)
  useLayoutEffect(() => {
    const ta = taRef.current
    if (!ta) return
    const prev = ta.placeholder
    if (!question.trim()) ta.placeholder = ''
    ta.style.height = '0px'
    ta.style.height = `${Math.max(96, Math.min(ta.scrollHeight, 320))}px`
    ta.placeholder = prev
  }, [question])

  return (
    <div
      className={`shrink-0 overflow-hidden rounded-xl border bg-white transition-colors dark:bg-zinc-900 ${
        error
          ? 'border-rose-300 dark:border-rose-900'
          : 'border-zinc-200 dark:border-zinc-800'
      }`}
    >
      <textarea
        ref={taRef}
        style={{ height: 96 }}
        value={question}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !disabled) {
            onAnalyze()
          }
        }}
        placeholder={'请粘贴带选项的题目，例如：\n\n这段文字意在说明：\nA. 选项一\nB. 选项二\nC. 选项三\nD. 选项四'}
        className="min-h-24 w-full resize-none border-0 bg-transparent p-4 text-sm leading-6 text-zinc-800 outline-none placeholder:text-zinc-400 dark:bg-transparent dark:text-zinc-100 dark:placeholder:text-zinc-500"
        spellCheck={false}
      />
      <div className="flex h-11 items-center justify-between gap-3 border-t border-zinc-100 px-4 dark:border-zinc-800">
        {/* 二级下拉：一级选供应商，二级选该供应商下的模型 */}
        <div className="flex min-w-0 items-center gap-1.5">
          {providers.length === 0 ? (
            <button
              onClick={onOpenSettings}
              className="text-xs text-zinc-400 underline-offset-2 transition-colors hover:text-zinc-600 hover:underline dark:text-zinc-500 dark:hover:text-zinc-300"
            >
              未配置 AI 供应商，点击配置
            </button>
          ) : (
            <>
              <div className="relative">
                <select
                  value={activeProviderId}
                  onChange={(e) => onProviderChange(e.target.value)}
                  disabled={loading}
                  aria-label="选择 AI 供应商"
                  className={selectCls}
                >
                  {providers.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name || '未命名'}
                    </option>
                  ))}
                </select>
                <ChevronDown
                  className="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-zinc-400"
                  aria-hidden="true"
                />
              </div>
              <div className="relative">
                <select
                  value={activeModel}
                  onChange={(e) => onModelChange(e.target.value)}
                  disabled={loading || !activeProvider || activeProvider.models.length === 0}
                  aria-label="选择模型"
                  className={selectCls}
                >
                  {activeProvider && activeProvider.models.length > 0 ? (
                    activeProvider.models.map((m) => (
                      <option key={m} value={m}>
                        {m}
                      </option>
                    ))
                  ) : (
                    <option value="">请先添加模型</option>
                  )}
                </select>
                <ChevronDown
                  className="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-zinc-400"
                  aria-hidden="true"
                />
              </div>
            </>
          )}
        </div>

        <div className="flex min-w-0 items-center gap-3">
          {error ? (
            <span className="inline-flex min-w-0 items-center gap-1 text-xs text-rose-500 dark:text-rose-400">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              <span className="truncate">{error}</span>
            </span>
          ) : pasteMsg ? (
            <span className="text-xs text-rose-500 dark:text-rose-400">{pasteMsg}</span>
          ) : (
            <span className="tnum text-xs text-zinc-400 dark:text-zinc-500">
              {question.trim().length} 字
            </span>
          )}
          <button
            onClick={() => void handlePaste()}
            disabled={loading}
            title="清空当前内容，粘贴剪贴板"
            className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg border border-zinc-200 bg-white px-3 text-[13px] font-medium text-zinc-600 transition-all hover:border-zinc-300 hover:bg-zinc-50 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-600 dark:hover:bg-zinc-800"
          >
            <ClipboardPaste className="h-3.5 w-3.5" aria-hidden="true" />
            粘贴
          </button>
          {loading ? (
            <button
              onClick={onCancel}
              className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg border border-rose-300 bg-rose-50 px-4 text-[13px] font-medium text-rose-600 transition-all hover:bg-rose-100 active:scale-[0.98] dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-300 dark:hover:bg-rose-950/70"
            >
              <CircleStop className="h-3.5 w-3.5" aria-hidden="true" />
              中止
            </button>
          ) : (
            <button
              onClick={onAnalyze}
              disabled={disabled}
              className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg bg-zinc-900 px-4 text-[13px] font-medium text-white transition-all hover:bg-zinc-700 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
            >
              <Send className="h-3.5 w-3.5" aria-hidden="true" />
              开始分析
              <kbd className="ml-1 hidden rounded border border-white/25 px-1 text-[10px] font-normal leading-4 text-white/70 lg:inline dark:border-zinc-900/20 dark:text-zinc-900/60">
                Ctrl ⏎
              </kbd>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
