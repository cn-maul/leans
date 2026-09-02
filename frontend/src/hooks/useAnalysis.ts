import { useCallback, useState } from 'react'
import * as api from '../api/client'
import type { AnalysisResult } from '../types/analysis'

export type AnalysisStage = 'idle' | 'running' | 'done'

export function useAnalysis() {
  const [result, setResult] = useState<AnalysisResult | null>(null)
  const [stage, setStage] = useState<AnalysisStage>('idle')
  const [error, setError] = useState('')

  const run = useCallback(async (subject: string, question: string) => {
    setStage('running')
    setError('')
    try {
      const res = await api.analyzeQuestion(subject, question)
      setResult(res)
      setStage('done')
      return res
    } catch (e) {
      setError(e instanceof Error ? e.message : '分析失败')
      setStage('idle')
      return null
    }
  }, [])

  const reset = useCallback(() => {
    setResult(null)
    setStage('idle')
    setError('')
  }, [])

  return { result, stage, error, run, reset, setResult }
}
