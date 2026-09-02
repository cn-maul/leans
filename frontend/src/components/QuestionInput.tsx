interface Props {
  question: string
  onChange: (val: string) => void
  onAnalyze: () => void
  loading: boolean
  hasSubject: boolean
}

export default function QuestionInput({ question, onChange, onAnalyze, loading, hasSubject }: Props) {
  const disabled = loading || !question.trim() || !hasSubject

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 shadow-sm">
      <div className="px-5 py-3 border-b border-gray-100 dark:border-gray-800 flex items-center justify-between">
        <h3 className="font-semibold text-gray-800 dark:text-gray-100 text-sm">粘贴题目</h3>
        {!hasSubject && (
          <span className="text-xs text-amber-600 dark:text-amber-400">请先在左侧选择科目</span>
        )}
      </div>
      <div className="p-5">
        <textarea
          value={question}
          onChange={(e) => onChange(e.target.value)}
          placeholder={'请粘贴带选项的题目，例如：\n\n这段文字意在说明：\nA. 选项一\nB. 选项二\nC. 选项三\nD. 选项四'}
          className="w-full h-64 p-4 border border-gray-200 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 rounded-lg text-sm leading-relaxed resize-none focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-shadow"
          spellCheck={false}
        />
        <div className="mt-3 flex items-center justify-between">
          <span className="text-xs text-gray-400 dark:text-gray-500">
            {question.trim().length} 字
          </span>
          <button
            onClick={onAnalyze}
            disabled={disabled}
            className="px-6 py-2.5 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 disabled:bg-gray-300 dark:disabled:bg-gray-700 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? '分析中…' : '开始分析'}
          </button>
        </div>
      </div>
    </div>
  )
}
