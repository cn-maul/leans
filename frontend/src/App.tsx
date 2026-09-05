import { useMemo, useState } from 'react'
import { CheckCircle2, Trash2 } from 'lucide-react'
import { fetchHistoryItem } from './api/client'
import type { AnalysisResult, Highlight, HistoryItem } from './types/analysis'
import { useAnalysis } from './hooks/useAnalysis'
import { useHistory } from './hooks/useHistory'
import { useSettings } from './hooks/useSettings'
import { useSubjects } from './hooks/useSubjects'
import { useTheme } from './hooks/useTheme'
import TopBar, { type View } from './components/TopBar'
import QuestionInput from './components/QuestionInput'
import {
  AnnotationPanel,
  ResultPills,
  TechniquePanel,
} from './components/AnalysisResult'
import StatsPage from './components/StatsPage'
import SettingsModal from './components/SettingsModal'
import HistoryPanel from './components/HistoryPanel'
import AnnotatedQuestion from './components/AnnotatedQuestion'
import Drawer from './components/Drawer'

function App() {
  const { subjects } = useSubjects()

  // 讲义默认锁定第一门（当前仅言语理解），不再提供切换入口。
  const selectedSubject = subjects[0]?.id || ''
  const lectureName = subjects[0]?.name || '言语理解'

  const [question, setQuestion] = useState('')
  const {
    result,
    stage,
    error: analysisError,
    run,
    setResult,
    partial,
    liveModel,
    liveFirstTokenMS,
  } = useAnalysis()
  const analysisLoading = stage === 'running'

  const { items: history, reload, clear } = useHistory()
  const { settings, saving, testing, error: settingsError, save, test } = useSettings()
  const { theme, toggle } = useTheme()

  const [view, setView] = useState<View>('analyze')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [toast, setToast] = useState('')
  const [validationError, setValidationError] = useState('')

  const handleAnalyze = async () => {
    const trimmed = question.trim()
    if (!selectedSubject) {
      setValidationError('讲义尚未加载完成，请稍后再试')
      return
    }
    if (trimmed.length < 6) {
      setValidationError('题目内容过短，请粘贴完整题干和选项')
      return
    }

    setValidationError('')
    await run(selectedSubject, trimmed)
    void reload()
  }

  // 历史回填：取完整记录（含结果 JSON），还原题目与分析结果。
  const handleHistorySelect = async (item: HistoryItem) => {
    try {
      const full = await fetchHistoryItem(item.id)
      setQuestion(full.question)
      setValidationError('')
      if (full.result) {
        try {
          setResult(JSON.parse(full.result) as AnalysisResult)
        } catch {
          setResult(null)
        }
      } else {
        setResult(null)
      }
      setView('analyze')
      setHistoryOpen(false)
      setToast('已恢复历史记录')
    } catch {
      setValidationError('历史记录读取失败，请稍后再试')
    }
  }

  const handleSaveSettings = async (s: typeof settings) => {
    try {
      await save(s)
      setSettingsOpen(false)
      setToast('设置已保存')
    } catch {
      /* 错误已在 useSettings 中记录 */
    }
  }

  // 合并所有标注：highlights + 规则命中的 marks（旧记录的绿色规则标注）。
  const allHighlights = useMemo<Highlight[]>(() => {
    if (!result) return []
    const merged = [...result.highlights]
    for (const rule of result.rules ?? []) {
      for (const m of rule.marks ?? []) {
        if (!merged.some((x) => x.text === m.text && x.location === m.location)) {
          merged.push(m)
        }
      }
    }
    return merged
  }, [result])

  const error = validationError || analysisError
  const hasResult = Boolean(result) && !analysisLoading

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      <TopBar
        lectureName={lectureName}
        view={view}
        onViewChange={setView}
        onOpenHistory={() => setHistoryOpen(true)}
        onOpenSettings={() => setSettingsOpen(true)}
        theme={theme}
        onToggleTheme={toggle}
        running={analysisLoading}
      />

      <main className="mx-auto flex min-h-0 w-full max-w-[1680px] flex-1 gap-4 p-4">
        {view === 'stats' ? (
          <StatsPage />
        ) : (
          <>
            {/* 左栏：固定视口高度，上半紧凑输入，下半标注视图为主 */}
            <section className="flex h-full w-[57%] min-w-0 flex-col gap-3">
              <QuestionInput
                question={question}
                onChange={setQuestion}
                onAnalyze={() => void handleAnalyze()}
                loading={analysisLoading}
                hasSubject={Boolean(selectedSubject)}
                error={error}
              />
              <AnnotatedQuestion
                question={question}
                highlights={allHighlights}
                active={hasResult}
              />
            </section>

            {/* 右栏：三胶囊 + 技巧应用 + 注释和思路 */}
            <section className="flex h-full min-w-0 flex-1 flex-col gap-3">
              <ResultPills result={result} loading={analysisLoading} partial={partial} />
              <TechniquePanel result={result} loading={analysisLoading} partial={partial} />
              <AnnotationPanel
                result={result}
                loading={analysisLoading}
                partial={partial}
                firstTokenMS={liveFirstTokenMS}
                model={liveModel}
              />
            </section>
          </>
        )}
      </main>

      {/* 历史抽屉 */}
      <Drawer
        open={historyOpen}
        title="分析历史"
        onClose={() => setHistoryOpen(false)}
        footer={
          history.length > 0 ? (
            <button
              onClick={() => void clear()}
              className="inline-flex w-full items-center justify-center gap-1.5 rounded-lg py-2 text-sm text-rose-500 transition-colors hover:bg-rose-50 dark:hover:bg-rose-950/40"
            >
              <Trash2 className="h-4 w-4" aria-hidden="true" />
              清空历史记录
            </button>
          ) : undefined
        }
      >
        <HistoryPanel items={history} onSelect={(item) => void handleHistorySelect(item)} />
      </Drawer>

      <SettingsModal
        open={settingsOpen}
        settings={settings}
        saving={saving}
        testing={testing}
        error={settingsError}
        onClose={() => setSettingsOpen(false)}
        onSave={(s) => void handleSaveSettings(s)}
        onTest={test}
      />

      {/* 轻提示 */}
      {toast && (
        <div className="animate-fade-up fixed bottom-6 left-1/2 z-[60] flex -translate-x-1/2 items-center gap-2 rounded-full bg-zinc-900 py-2.5 pr-5 pl-4 text-sm text-white shadow-lg dark:bg-zinc-100 dark:text-zinc-900">
          <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
          {toast}
        </div>
      )}
    </div>
  )
}

export default App
