import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'
import type { AISettings } from '../types/analysis'

export function useSettings() {
  const [settings, setSettings] = useState<AISettings>({
    provider: 'openai',
    api_key: '',
    base_url: '',
    model: '',
  })
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .fetchSettings()
      .then(setSettings)
      .catch((e) => setError(e instanceof Error ? e.message : '加载设置失败'))
  }, [])

  const save = useCallback(async (next: AISettings) => {
    setSaving(true)
    setError('')
    try {
      const saved = await api.saveSettings(next)
      setSettings(saved)
      return saved
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存设置失败'
      setError(msg)
      throw e
    } finally {
      setSaving(false)
    }
  }, [])

  const test = useCallback(async () => {
    setTesting(true)
    setError('')
    try {
      const res = await api.testConnection()
      return res.ok
    } catch (e) {
      const msg = e instanceof Error ? e.message : '测试连接失败'
      setError(msg)
      return false
    } finally {
      setTesting(false)
    }
  }, [])

  return { settings, saving, testing, error, save, test }
}
