import { useMemo, useRef, useState } from 'react'
import { fetchHistoryItem } from './api/client'
import type { AnalysisResult, HistoryItem, Highlight } from './types/analysis'
import { useAnalysis } from './hooks/useAnalysis'
import { useHistory } from './hooks/useHistory'
import { useSettings } from './hooks/useSettings'
import { useSubjectContent } from './hooks/useSubjectContent'
import { useSubjects } from './hooks/useSubjects'
import { useTheme } from './hooks/useTheme'
import TopBar from './components/TopBar'
import QuestionInput from './components/QuestionInput'
import AnalysisResultPanel from './components/AnalysisResult'
import SubjectContentPanel from './components/SubjectContentPanel'
import SettingsModal from './components/SettingsModal'
import HistoryPanel from './components/HistoryPanel'
import AnnotatedQuestion from './components/AnnotatedQuestion'
import Drawer from './components/Drawer'

function App() {
  const { subjects } = useSubjects()
  const [selectedSubject, setSelectedSubject] = useState('')
  const { content, loading: contentLoading } = useSubjectContent(selectedSubject)
  const [question, setQuestion] = useState('')

  const { result, stage, error: analysisError, run, setResult } = useAnalysis()
  const analysisLoading = stage === 'running'

  const { items: history, reload: reloadHistory, clear: clearHistory } = useHistory()
  const { settings, saving, testing, error: settingsError, save, test } = useSettings()
  const { theme, toggle } = useTheme()

  const [settingsOpen, setSettingsOpen] = useState(false)
  const [lectureOpen, setLectureOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [toast, setToast] = useState('')

  // 记录最近一次设置错误，用于在弹窗内显示。
  const lastSettingsError = useRef('')
  if (settingsError) lastSettingsError.current = settingsError

  const handleAnalyze = async () => {
    if (!selectedSubject || !question.trim()) return
    await run(selectedSubject, question)
    void reloadHistory()
  }

  // 历史回填：取完整记录（含结果 JSON），还原科目、题目与分析结果。
  const handleHistorySelect = async (item: HistoryItem) => {
    try {
      const full = await fetchHistoryItem(item.id)
      setSelectedSubject(full.subject)
      setQuestion(full.question)
      if (full.result) {
        try {
          setResult(JSON.parse(full.result) as AnalysisResult)
        } catch {
          /* 旧数据无 result 时只回填题目 */
        }
      } else {
        setResult(null)
      }
      setHistoryOpen(false)
    } catch {
      /* 静默：历史项可能已被清空 */
    }
  }

  const handleSaveSettings = async (s: typeof settings) => {
    try {
      await save(s)
      lastSettingsError.current = ''
      setSettingsOpen(false)
      setToast('设置已保存')
    } catch {
      /* 错误已在 useSettings 中记录 */
    }
  }

  // 合并所有标注：highlights + 规则命中的 marks（保证左栏标注与右栏模块一致）。
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

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex flex-col">
      <TopBar
        subjects={subjects}
        selectedSubject={selectedSubject}
        onSelectSubject={setSelectedSubject}
        onOpenLecture={() => setLectureOpen(true)}
        onOpenHistory={() => setHistoryOpen(true)}
        onOpenSettings={() => {
          lastSettingsError.current = ''
          setSettingsOpen(true)
        }}
        theme={theme}
        onToggleTheme={toggle}
      />

      <main className="flex-1 max-w-[1440px] w-full mx-auto p-6">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* 左栏：贴题 + 标注 */}
          <div className="lg:col-span-7 space-y-4">
            <QuestionInput
              question={question}
              onChange={setQuestion}
              onAnalyze={() => void handleAnalyze()}
              loading={analysisLoading}
              hasSubject={!!selectedSubject}
            />

            {analysisError && (
              <div className="p-4 bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-xl text-sm">
                {analysisError}
              </div>
            )}

            {result && !analysisLoading && (
              <AnnotatedQuestion question={question} highlights={allHighlights} />
            )}
          </div>

          {/* 右栏：分析结果 */}
          <div className="lg:col-span-5">
            <AnalysisResultPanel result={result} loading={analysisLoading} />
          </div>
        </div>
      </main>

      {/* 讲义抽屉 */}
      <Drawer open={lectureOpen} title="讲义内容" onClose={() => setLectureOpen(false)}>
        <div className="p-4">
          <SubjectContentPanel content={content} loading={contentLoading} />
        </div>
      </Drawer>

      {/* 历史抽屉 */}
      <Drawer
        open={historyOpen}
        title="分析历史"
        onClose={() => setHistoryOpen(false)}
        footer={
          history.length > 0 ? (
            <button
              onClick={() => void clearHistory()}
              className="w-full py-2 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
            >
              清空历史记录
            </button>
          ) : undefined
        }
      >
        <div className="p-4">
          <HistoryPanel
            items={history}
            onSelect={(item) => void handleHistorySelect(item)}
          />
        </div>
      </Drawer>

      <SettingsModal
        open={settingsOpen}
        settings={settings}
        saving={saving}
        testing={testing}
        error={settingsError || lastSettingsError.current}
        onClose={() => setSettingsOpen(false)}
        onSave={(s) => void handleSaveSettings(s)}
        onTest={test}
      />

      {/* 轻提示 */}
      {toast && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-[60] px-4 py-2 bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-sm rounded-lg shadow-lg">
          {toast}
        </div>
      )}
    </div>
  )
}

export default App
