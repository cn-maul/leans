import { useEffect, useState } from 'react'
import type { AISettings } from '../types/analysis'

interface Props {
  open: boolean
  settings: AISettings
  saving: boolean
  testing: boolean
  error: string
  onClose: () => void
  onSave: (s: AISettings) => void
  onTest: () => Promise<boolean>
}

const PROVIDERS: Array<{ value: string; label: string; base_url: string; model: string }> = [
  { value: 'openai', label: 'OpenAI', base_url: 'https://api.openai.com/v1', model: 'gpt-4o' },
  {
    value: 'deepseek',
    label: 'DeepSeek',
    base_url: 'https://api.deepseek.com/v1',
    model: 'deepseek-chat',
  },
  {
    value: 'qwen',
    label: '通义千问',
    base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    model: 'qwen-plus',
  },
  { value: 'custom', label: '自定义', base_url: '', model: '' },
]

// maskKey 对已保存的 API Key 做脱敏展示（仅用于编辑提示，不覆盖用户新输入）。
export function maskKey(key: string): string {
  if (!key) return ''
  if (key.length <= 8) return '••••••••'
  return `${key.slice(0, 4)}••••${key.slice(-4)}`
}

export default function SettingsModal({
  open,
  settings,
  saving,
  testing,
  error,
  onClose,
  onSave,
  onTest,
}: Props) {
  const [form, setForm] = useState<AISettings>(settings)
  const [keyTouched, setKeyTouched] = useState(false)
  const [testOk, setTestOk] = useState<boolean | null>(null)

  // 打开时重置为最新设置。
  useEffect(() => {
    if (open) {
      setForm(settings)
      setKeyTouched(false)
      setTestOk(null)
    }
  }, [open, settings])

  if (!open) return null

  const set = (k: keyof AISettings, v: string) => {
    setForm((f) => ({ ...f, [k]: v }))
    if (k === 'api_key') setKeyTouched(true)
  }

  const handleProvider = (value: string) => {
    const p = PROVIDERS.find((x) => x.value === value)
    setForm((f) => ({
      ...f,
      provider: value,
      base_url: p?.base_url || f.base_url,
      model: p?.model || f.model,
    }))
  }

  // 提交时：未改动的 API Key 用原值；改动过才用新值。
  const handleSave = () => {
    onSave({ ...form, api_key: keyTouched ? form.api_key : settings.api_key })
  }

  const handleTest = async () => {
    setTestOk(null)
    const ok = await onTest()
    setTestOk(ok)
  }

  const inputCls =
    'w-full p-2 border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-shadow'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="bg-white dark:bg-gray-900 rounded-2xl shadow-2xl w-[560px] max-w-full">
        <div className="px-6 py-4 border-b border-gray-100 dark:border-gray-800 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100">AI 设置</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 text-xl leading-none transition-colors"
            aria-label="关闭"
          >
            &times;
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              服务商
            </label>
            <select value={form.provider} onChange={(e) => handleProvider(e.target.value)} className={inputCls}>
              {PROVIDERS.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Base URL
            </label>
            <input
              value={form.base_url}
              onChange={(e) => set('base_url', e.target.value)}
              placeholder="https://api.openai.com/v1"
              className={inputCls}
              spellCheck={false}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              模型
            </label>
            <input
              value={form.model}
              onChange={(e) => set('model', e.target.value)}
              placeholder="gpt-4o / deepseek-chat"
              className={inputCls}
              spellCheck={false}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              API Key
            </label>
            <input
              type="password"
              value={form.api_key}
              onChange={(e) => set('api_key', e.target.value)}
              placeholder={settings.api_key ? `${maskKey(settings.api_key)}（留空则保留原 Key）` : 'sk-...'}
              className={inputCls}
              autoComplete="off"
              spellCheck={false}
            />
            {settings.api_key && !keyTouched && (
              <p className="text-[11px] text-gray-400 dark:text-gray-500 mt-1">
                已保存 Key：{maskKey(settings.api_key)}，如需更换请重新输入。
              </p>
            )}
          </div>

          {error && (
            <div className="p-3 bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-lg text-sm">
              {error}
            </div>
          )}
          {testOk === true && !error && (
            <div className="p-3 bg-green-50 dark:bg-green-900/30 text-green-700 dark:text-green-300 rounded-lg text-sm">
              ✓ 连接成功，配置可用
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-gray-100 dark:border-gray-800 flex items-center justify-between gap-3">
          <button
            onClick={() => void handleTest()}
            disabled={testing || saving}
            className="px-4 py-2 text-sm text-primary-600 dark:text-primary-400 border border-primary-300 dark:border-primary-700 rounded-lg hover:bg-primary-50 dark:hover:bg-primary-900/30 disabled:opacity-50 transition-colors"
          >
            {testing ? '测试中…' : '测试连接'}
          </button>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
            >
              取消
            </button>
            <button
              onClick={handleSave}
              disabled={saving || testing}
              className="px-5 py-2 text-sm text-white bg-primary-600 rounded-lg hover:bg-primary-700 disabled:bg-gray-300 dark:disabled:bg-gray-700 transition-colors"
            >
              {saving ? '保存中…' : '保存'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
