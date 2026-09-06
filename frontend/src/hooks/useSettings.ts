import { useCallback, useEffect, useState } from 'react'
import * as api from '../api/client'
import type { AISettings } from '../types/analysis'

const EMPTY_SETTINGS: AISettings = {
  active_provider_id: '',
  active_model: '',
  providers: [],
}

export function useSettings() {
  const [settings, setSettings] = useState<AISettings>(EMPTY_SETTINGS)
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

  // select 切换主界面二级下拉的供应商/模型（model 为空则由后端自动选）。
  const select = useCallback(async (providerId: string, model: string) => {
    const saved = await api.setActiveSelection(providerId, model)
    setSettings(saved)
    return saved
  }, [])

  const test = useCallback(
    async (
      provider?: { name?: string; base_url: string; api_key: string; protocol?: string },
      model?: string,
    ) => {
      setTesting(true)
      setError('')
      try {
        const res = await api.testConnection(provider, model)
        return res.ok
      } catch (e) {
        const msg = e instanceof Error ? e.message : '测试连接失败'
        setError(msg)
        return false
      } finally {
        setTesting(false)
      }
    },
    [],
  )

  return { settings, saving, testing, error, save, select, test }
}
