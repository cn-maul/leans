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
  'h-8 max-w-[150px] appearance-none rounded-chip border border-hairline bg-surface py-0 pl-2.5 pr-7 text-xs text-body transition-colors duration-150 ease-quart hover:border-hint focus:border-accent focus:outline-none disabled:cursor-not-allowed disabled:opacity-40'

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
      className={`shrink-0 overflow-hidden rounded-card bg-surface shadow-card transition-shadow duration-200 ease-quart ${
        error ? 'ring-[1.5px] ring-danger' : ''
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
        className="min-h-24 w-full resize-none border-0 bg-transparent p-4 text-[15px] leading-7 text-body outline-none placeholder:text-ghost"
        spellCheck={false}
      />
      <div className="flex h-12 items-center justify-between gap-3 border-t border-hairline px-4">
        {/* 二级下拉：一级选供应商，二级选该供应商下的模型 */}
        <div className="flex min-w-0 items-center gap-1.5">
          {providers.length === 0 ? (
            <button
              onClick={onOpenSettings}
              className="text-xs text-quiet underline-offset-2 transition-colors duration-150 ease-quart hover:text-accent hover:underline"
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
                  className="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-ghost"
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
                  className="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-ghost"
                  aria-hidden="true"
                />
              </div>
            </>
          )}
        </div>

        <div className="flex min-w-0 items-center gap-3">
          {error ? (
            <span className="inline-flex min-w-0 items-center gap-1 text-xs text-danger">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              <span className="truncate">{error}</span>
            </span>
          ) : pasteMsg ? (
            <span className="text-xs text-danger">{pasteMsg}</span>
          ) : (
            <span className="tnum text-xs text-quiet">{question.trim().length} 字</span>
          )}
          <button
            onClick={() => void handlePaste()}
            disabled={loading}
            title="清空当前内容，粘贴剪贴板"
            className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-pill bg-fill px-3.5 text-[13px] font-medium text-body transition-[background-color,transform] duration-150 ease-quart hover:bg-fill-strong active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-40"
          >
            <ClipboardPaste className="h-3.5 w-3.5" aria-hidden="true" />
            粘贴
          </button>
          {loading ? (
            <button
              onClick={onCancel}
              className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-pill bg-danger-tint px-4 text-[13px] font-medium text-danger transition-transform duration-150 ease-quart active:scale-[0.97]"
            >
              <CircleStop className="h-3.5 w-3.5" aria-hidden="true" />
              中止
            </button>
          ) : (
            <button
              onClick={onAnalyze}
              disabled={disabled}
              className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-pill bg-accent px-4 text-[13px] font-medium text-white transition-[background-color,transform,opacity] duration-150 ease-quart hover:bg-accent-deep active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100"
            >
              <Send className="h-3.5 w-3.5" aria-hidden="true" />
              开始分析
              <kbd className="ml-1 hidden rounded-chip border border-white/30 px-1 text-[10px] font-normal leading-4 text-white/75 lg:inline">
                Ctrl ⏎
              </kbd>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
