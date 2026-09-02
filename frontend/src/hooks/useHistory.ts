import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'
import type { HistoryItem } from '../types/analysis'

export function useHistory() {
  const [items, setItems] = useState<HistoryItem[]>([])
  const [error, setError] = useState('')

  const reload = useCallback(async () => {
    try {
      setItems(await api.fetchHistory())
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载历史失败')
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  const clear = useCallback(async () => {
    try {
      await api.clearHistory()
      setItems([])
      setError('')
    } catch (e) {
      setError(e instanceof Error ? e.message : '清空历史失败')
    }
  }, [])

  return { items, error, reload, clear }
}
