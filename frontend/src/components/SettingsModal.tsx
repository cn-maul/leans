import { useEffect, useState } from 'react'
import { AlertCircle, CheckCircle2, ChevronDown, KeyRound, Plug, Save, X } from 'lucide-react'
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

  const handleSave = () => {
    onSave({ ...form, api_key: keyTouched ? form.api_key : settings.api_key })
  }

  const handleTest = async () => {
    setTestOk(null)
    const ok = await onTest()
    setTestOk(ok)
  }

  const inputCls =
    'w-full rounded-lg border border-zinc-200 bg-white px-3 py-2.5 text-sm text-zinc-800 transition-colors placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-2 focus:ring-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-950/60 dark:text-zinc-100 dark:placeholder:text-zinc-500 dark:focus:ring-zinc-100/10'

  return (
    <div
      className="animate-fade-in fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-label="AI 设置"
    >
      <div className="animate-fade-up w-[560px] max-w-full overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-900">
        <div className="flex h-14 items-center justify-between border-b border-zinc-100 px-6 dark:border-zinc-800">
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">AI 设置</h2>
          <button
            onClick={onClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
            aria-label="关闭"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        <div className="space-y-4 p-6">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
              服务商
            </label>
            <div className="relative">
              <select
                value={form.provider}
                onChange={(e) => handleProvider(e.target.value)}
                className={`${inputCls} appearance-none pr-9`}
              >
                {PROVIDERS.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </select>
              <ChevronDown
                className="pointer-events-none absolute top-1/2 right-3 h-4 w-4 -translate-y-1/2 text-zinc-400"
                aria-hidden="true"
              />
            </div>
          </div>

          <div>
            <label className="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
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
            <label className="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
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
            <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-zinc-700 dark:text-zinc-300">
              <KeyRound className="h-3.5 w-3.5 text-zinc-400" aria-hidden="true" />
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
              <p className="mt-1.5 text-[11px] text-zinc-400 dark:text-zinc-500">
                已保存 Key：{maskKey(settings.api_key)}，如需更换请重新输入。
              </p>
            )}
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2.5 text-sm text-rose-700 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-300">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              {error}
            </div>
          )}
          {testOk === true && !error && (
            <div className="flex items-start gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-sm text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              连接成功，配置可用
            </div>
          )}
        </div>

        <div className="flex items-center justify-between gap-3 border-t border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <button
            onClick={() => void handleTest()}
            disabled={testing || saving}
            className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-zinc-200 px-3.5 text-sm text-zinc-600 transition-colors hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
          >
            {testing ? (
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-zinc-400 border-t-transparent" />
            ) : (
              <Plug className="h-4 w-4" aria-hidden="true" />
            )}
            {testing ? '测试中…' : '测试连接'}
          </button>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="inline-flex h-9 items-center rounded-lg border border-zinc-200 px-4 text-sm text-zinc-600 transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
            >
              取消
            </button>
            <button
              onClick={handleSave}
              disabled={saving || testing}
              className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-zinc-900 px-4 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
            >
              <Save className="h-4 w-4" aria-hidden="true" />
              {saving ? '保存中…' : '保存'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
