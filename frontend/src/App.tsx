import { useMemo, useState } from 'react'
import { CheckCircle2, Trash2 } from 'lucide-react'
import { fetchHistoryItem } from './api/client'
import type { AnalysisResult, Highlight, HistoryItem } from './types/analysis'
import { isSubjectiveName } from './types/analysis'
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
import ShenlunPage, { type RestorePayload } from './components/ShenlunPage'
import Drawer from './components/Drawer'

function App() {
  const { subjects } = useSubjects()

  // 按科目种类各选各的讲义：言语页锁客观题，申论页锁主观题（申论）。
  // 不再依赖 subjects[0]，避免加入申论后默认排序改变而劫持言语页。
  const choiceSubject = subjects.find((s) => !isSubjectiveName(s.name))
  const shenlunSubject = subjects.find((s) => isSubjectiveName(s.name))
  const selectedSubject = choiceSubject?.id || ''
  const lectureName = choiceSubject?.name || '言语理解'

  const [question, setQuestion] = useState('')
  const {
    result,
    stage,
    error: analysisError,
    run,
    cancel: cancelAnalysis,
    setResult,
    partial,
    liveModel,
    liveFirstTokenMS,
    liveStartedAt,
  } = useAnalysis()
  const analysisLoading = stage === 'running'

  const { items: history, reload, clear } = useHistory()
  const { settings, saving, error: settingsError, save, select, test } = useSettings()
  const { theme, toggle } = useTheme()

  const [view, setView] = useState<View>('analyze')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [toast, setToast] = useState('')
  const [validationError, setValidationError] = useState('')
  // 申论页独立状态：历史恢复载荷 + 本页加载态（供 TopBar 进度条合并显示）。
  const [shenlunRestore, setShenlunRestore] = useState<RestorePayload | null>(null)
  const [shenlunRunning, setShenlunRunning] = useState(false)

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

  // 历史回填：取完整记录（含结果 JSON），按科目种类落到对应页面。
  const handleHistorySelect = async (item: HistoryItem) => {
    try {
      const full = await fetchHistoryItem(item.id)
      setValidationError('')
      const parsed = full.result
        ? (() => {
            try {
              return JSON.parse(full.result) as AnalysisResult
            } catch {
              return null
            }
          })()
        : null

      if (isSubjectiveName(full.subject)) {
        // 申论记录 → 灌回申论页自己的状态。
        setShenlunRestore({
          material: full.question,
          userAnswer: full.user_answer || undefined,
          result: parsed,
          nonce: Date.now(),
        })
        setView('shenlun')
      } else {
        setQuestion(full.question)
        setResult(parsed)
        setView('analyze')
      }
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

  // 二级下拉切换：供应商切换时模型由后端自动选中该供应商的可用模型。
  const handleProviderChange = (id: string) => {
    select(id, '').catch((e) => {
      setValidationError(e instanceof Error ? e.message : '切换供应商失败')
    })
  }
  const handleModelChange = (model: string) => {
    select(settings.active_provider_id, model).catch((e) => {
      setValidationError(e instanceof Error ? e.message : '切换模型失败')
    })
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
    <div className="flex h-screen flex-col overflow-hidden bg-ground text-ink">
      <TopBar
        lectureName={view === 'shenlun' ? shenlunSubject?.name || '申论' : lectureName}
        view={view}
        onViewChange={setView}
        onOpenHistory={() => setHistoryOpen(true)}
        onOpenSettings={() => setSettingsOpen(true)}
        theme={theme}
        onToggleTheme={toggle}
        running={analysisLoading || shenlunRunning}
      />

      <main className="mx-auto flex min-h-0 w-full max-w-[1680px] flex-1 gap-4 p-4 max-lg:flex-col max-lg:overflow-y-auto">
        {view === 'stats' ? (
          <StatsPage />
        ) : view === 'shenlun' ? (
          <ShenlunPage
            subjectId={shenlunSubject?.id || ''}
            hasSubject={Boolean(shenlunSubject)}
            providers={settings.providers}
            activeProviderId={settings.active_provider_id}
            activeModel={settings.active_model}
            onProviderChange={handleProviderChange}
            onModelChange={handleModelChange}
            onOpenSettings={() => setSettingsOpen(true)}
            onAnalyzed={() => void reload()}
            restore={shenlunRestore}
            onRunningChange={setShenlunRunning}
          />
        ) : (
          <>
            {/* 左栏：上半紧凑输入，下半标注视图为主 */}
            <section className="flex h-full w-[57%] min-w-0 flex-col gap-3 max-lg:h-auto max-lg:w-full">
              <QuestionInput
                question={question}
                onChange={setQuestion}
                onAnalyze={() => void handleAnalyze()}
                onCancel={cancelAnalysis}
                loading={analysisLoading}
                hasSubject={Boolean(selectedSubject)}
                error={error}
                providers={settings.providers}
                activeProviderId={settings.active_provider_id}
                activeModel={settings.active_model}
                onProviderChange={handleProviderChange}
                onModelChange={handleModelChange}
                onOpenSettings={() => setSettingsOpen(true)}
              />
              <AnnotatedQuestion
                question={question}
                highlights={allHighlights}
                active={hasResult}
              />
            </section>

            {/* 右栏：三胶囊 + 技巧应用 + 注释和思路 */}
            <section className="flex h-full min-w-0 flex-1 flex-col gap-3 max-lg:h-auto">
              <ResultPills result={result} loading={analysisLoading} partial={partial} />
              <TechniquePanel result={result} loading={analysisLoading} partial={partial} />
              <AnnotationPanel
                result={result}
                loading={analysisLoading}
                partial={partial}
                firstTokenMS={liveFirstTokenMS}
                model={liveModel}
                startedAt={liveStartedAt}
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
              className="inline-flex h-11 w-full items-center justify-center gap-1.5 rounded-sheet text-sm font-medium text-danger transition-colors duration-150 ease-quart hover:bg-danger-tint active:scale-[0.99]"
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
        error={settingsError}
        onClose={() => setSettingsOpen(false)}
        onSave={(s) => void handleSaveSettings(s)}
        onTest={test}
      />

      {/* 轻提示 */}
      {toast && (
        <div className="animate-rise fixed bottom-6 left-1/2 z-[60] flex items-center gap-2 rounded-pill bg-ink py-2.5 pr-5 pl-4 text-sm text-surface shadow-overlay">
          <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
          {toast}
        </div>
      )}
    </div>
  )
}

export default App
