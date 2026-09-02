import { useEffect, useState } from 'react'
import * as api from '../api/client'
import type { SubjectContent } from '../types/analysis'

export function useSubjectContent(subjectId: string | null) {
  const [content, setContent] = useState<SubjectContent | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!subjectId) {
      setContent(null)
      setError('')
      return
    }
    let cancelled = false
    setLoading(true)
    setError('')
    api
      .fetchSubjectContent(subjectId)
      .then((c) => {
        if (!cancelled) setContent(c)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : '加载讲义失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [subjectId])

  return { content, loading, error }
}
