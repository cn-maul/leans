import { AlertCircle, Loader2, Send } from 'lucide-react'

interface Props {
  question: string
  onChange: (val: string) => void
  onAnalyze: () => void
  loading: boolean
  hasSubject: boolean
  error: string
}

// 紧凑输入卡：textarea 固定压缩高度，错误就地显示在底栏，替代页面顶部 banner。
export default function QuestionInput({
  question,
  onChange,
  onAnalyze,
  loading,
  hasSubject,
  error,
}: Props) {
  const disabled = loading || !question.trim() || !hasSubject

  return (
    <div
      className={`shrink-0 overflow-hidden rounded-xl border bg-white transition-colors dark:bg-zinc-900 ${
        error
          ? 'border-rose-300 dark:border-rose-900'
          : 'border-zinc-200 dark:border-zinc-800'
      }`}
    >
      <textarea
        value={question}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !disabled) {
            onAnalyze()
          }
        }}
        placeholder={'请粘贴带选项的题目，例如：\n\n这段文字意在说明：\nA. 选项一\nB. 选项二\nC. 选项三\nD. 选项四'}
        className="h-24 w-full resize-none border-0 bg-transparent p-4 text-sm leading-6 text-zinc-800 outline-none placeholder:text-zinc-400 dark:bg-transparent dark:text-zinc-100 dark:placeholder:text-zinc-500"
        spellCheck={false}
      />
      <div className="flex h-11 items-center justify-between gap-4 border-t border-zinc-100 px-4 dark:border-zinc-800">
        {error ? (
          <span className="inline-flex min-w-0 items-center gap-1 text-xs text-rose-500 dark:text-rose-400">
            <AlertCircle className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span className="truncate">{error}</span>
          </span>
        ) : (
          <span className="tnum text-xs text-zinc-400 dark:text-zinc-500">
            {question.trim().length} 字
          </span>
        )}
        <button
          onClick={onAnalyze}
          disabled={disabled}
          className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg bg-zinc-900 px-4 text-[13px] font-medium text-white transition-all hover:bg-zinc-700 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          {loading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
          ) : (
            <Send className="h-3.5 w-3.5" aria-hidden="true" />
          )}
          {loading ? '分析中…' : '开始分析'}
          <kbd className="ml-1 hidden rounded border border-white/25 px-1 text-[10px] font-normal leading-4 text-white/70 lg:inline dark:border-zinc-900/20 dark:text-zinc-900/60">
            Ctrl ⏎
          </kbd>
        </button>
      </div>
    </div>
  )
}
