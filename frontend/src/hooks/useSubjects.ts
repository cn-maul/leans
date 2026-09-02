import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'
import type { Subject } from '../types/analysis'

export function useSubjects() {
  const [subjects, setSubjects] = useState<Subject[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const reload = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      setSubjects(await api.fetchSubjects())
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载科目失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  return { subjects, loading, error, reload }
}
