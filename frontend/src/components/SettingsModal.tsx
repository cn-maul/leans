import { useEffect, useState } from 'react'
import {
  AlertCircle,
  Check,
  CheckCircle2,
  ChevronDown,
  KeyRound,
  Plus,
  Plug,
  RefreshCw,
  Save,
  Trash2,
  X,
} from 'lucide-react'
import type { AIProvider, AISettings } from '../types/analysis'
import { fetchProviderModels } from '../api/client'

interface Props {
  open: boolean
  settings: AISettings
  saving: boolean
  error: string
  onClose: () => void
  onSave: (s: AISettings) => void
  onTest: (
    provider: { name?: string; base_url: string; api_key: string; protocol?: string },
    model?: string,
  ) => Promise<boolean>
}

export function maskKey(key: string): string {
  if (!key) return ''
  if (key.length <= 8) return '••••••••'
  return `${key.slice(0, 4)}••••${key.slice(-4)}`
}

const inputCls =
  'w-full rounded-lg border border-zinc-200 bg-white px-3 py-2.5 text-sm text-zinc-800 transition-colors placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-2 focus:ring-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-950/60 dark:text-zinc-100 dark:placeholder:text-zinc-500 dark:focus:ring-zinc-100/10'

// 取值与后端 model 包协议常量一致。
const PROTOCOLS: Array<{ value: string; label: string }> = [
  { value: 'auto', label: '自动检测（推荐）' },
  { value: 'openai-chat', label: 'OpenAI 兼容（Chat Completions）' },
  { value: 'openai-responses', label: 'OpenAI Responses' },
  { value: 'anthropic', label: 'Anthropic Messages' },
]

// newId 生成供应商 ID；crypto 不可用时退化为时间戳。
function newId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `p-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

export default function SettingsModal({
  open,
  settings,
  saving,
  error,
  onClose,
  onSave,
  onTest,
}: Props) {
  const [draft, setDraft] = useState<AISettings>(settings)
  const [editingIdx, setEditingIdx] = useState(0)
  const [newModel, setNewModel] = useState('')
  const [available, setAvailable] = useState<string[]>([])
  const [fetching, setFetching] = useState(false)
  const [testing, setTesting] = useState(false)
  const [localError, setLocalError] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => {
    if (open) {
      setDraft(structuredClone(settings))
      const idx = Math.max(
        0,
        settings.providers.findIndex((p) => p.id === settings.active_provider_id),
      )
      setEditingIdx(settings.providers.length ? idx : 0)
      setNewModel('')
      setAvailable([])
      setLocalError('')
      setNotice('')
    }
  }, [open, settings])

  if (!open) return null

  const editing = draft.providers[editingIdx]

  const patchProvider = (patch: Partial<AIProvider>) => {
    setDraft((d) => ({
      ...d,
      providers: d.providers.map((p, i) => (i === editingIdx ? { ...p, ...patch } : p)),
    }))
  }

  // 切换编辑对象/新增时清空提示与上一个供应商的获取结果。
  const selectEditing = (i: number) => {
    setEditingIdx(i)
    setNewModel('')
    setAvailable([])
    setLocalError('')
    setNotice('')
  }

  const addProvider = () => {
    const p: AIProvider = {
      id: newId(),
      name: '',
      base_url: '',
      api_key: '',
      protocol: 'auto',
      models: [],
    }
    setDraft((d) => ({ ...d, providers: [...d.providers, p] }))
    selectEditing(draft.providers.length)
  }

  const removeProvider = () => {
    if (!editing) return
    if (!window.confirm(`确定删除供应商「${editing.name || '未命名'}」吗？`)) return
    setDraft((d) => {
      const providers = d.providers.filter((_, i) => i !== editingIdx)
      const activeGone = d.active_provider_id === editing.id
      return {
        ...d,
        providers,
        active_provider_id: activeGone ? '' : d.active_provider_id,
      }
    })
    setEditingIdx((i) => Math.max(0, Math.min(i, draft.providers.length - 2)))
    setAvailable([])
  }

  // addModel 把模型加入当前供应商列表；已存在时忽略。
  const addModel = (raw: string) => {
    const m = raw.trim()
    if (!m || !editing || editing.models.includes(m)) return
    const models = [...editing.models, m]
    patchProvider({ models })
    setDraft((d) =>
      d.active_provider_id === editing.id && !models.includes(d.active_model)
        ? { ...d, active_model: d.active_model || m }
        : d,
    )
  }

  const removeModel = (m: string) => {
    if (!editing) return
    patchProvider({ models: editing.models.filter((x) => x !== m) })
  }

  // handleFetchModels 在线拉取模型列表；结果进入待选区，由用户点选添加。
  const handleFetchModels = async () => {
    if (!editing) return
    setLocalError('')
    setNotice('')
    if (!editing.base_url.trim()) {
      setLocalError('请先填写 Base URL')
      return
    }
    setFetching(true)
    try {
      const { models } = await fetchProviderModels(
        editing.base_url.trim(),
        editing.api_key,
        editing.protocol || 'auto',
      )
      setAvailable(models)
      setNotice(
        models.length > 0 ? `已获取 ${models.length} 个模型，点击下方列表添加` : '服务端未返回任何模型',
      )
    } catch (e) {
      setAvailable([])
      setLocalError(e instanceof Error ? e.message : '获取模型失败')
    } finally {
      setFetching(false)
    }
  }

  const handleTest = async () => {
    if (!editing) return
    setLocalError('')
    setNotice('')
    setTesting(true)
    const ok = await onTest(
      {
        name: editing.name,
        base_url: editing.base_url,
        api_key: editing.api_key,
        protocol: editing.protocol,
      },
      editing.models[0],
    )
    setTesting(false)
    setNotice(ok ? '连接成功，配置可用' : '')
  }

  const handleSave = () => {
    onSave(draft)
  }

  return (
    <div
      className="animate-fade-in fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-label="AI 设置"
    >
      <div className="animate-fade-up flex max-h-[90vh] w-[600px] max-w-full flex-col overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-900">
        <div className="flex h-14 shrink-0 items-center justify-between border-b border-zinc-100 px-6 dark:border-zinc-800">
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            AI 设置 · 供应商
          </h2>
          <button
            onClick={onClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-700 dark:text-zinc-500 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
            aria-label="关闭"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-5 overflow-y-auto p-6">
          {/* 供应商列表 */}
          <div>
            <div className="mb-2 flex items-center justify-between">
              <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">供应商</span>
              <button
                onClick={addProvider}
                className="inline-flex h-7 items-center gap-1 rounded-lg border border-dashed border-zinc-300 px-2 text-xs text-zinc-500 transition-colors hover:border-zinc-400 hover:text-zinc-700 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-zinc-600 dark:hover:text-zinc-200"
              >
                <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                添加
              </button>
            </div>
            {draft.providers.length === 0 ? (
              <p className="rounded-lg border border-dashed border-zinc-200 px-3 py-4 text-center text-xs text-zinc-400 dark:border-zinc-700 dark:text-zinc-500">
                尚未添加供应商，点击「添加」开始配置
              </p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {draft.providers.map((p, i) => (
                  <button
                    key={p.id}
                    onClick={() => selectEditing(i)}
                    className={`inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-xs transition-colors ${
                      i === editingIdx
                        ? 'border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900'
                        : 'border-zinc-200 text-zinc-600 hover:border-zinc-400 dark:border-zinc-700 dark:text-zinc-300 dark:hover:border-zinc-500'
                    }`}
                  >
                    {settings.active_provider_id === p.id && (
                      <span
                        className="h-1.5 w-1.5 rounded-full bg-emerald-500"
                        aria-hidden="true"
                      />
                    )}
                    {p.name || '未命名'}
                  </button>
                ))}
              </div>
            )}
            <p className="mt-1.5 text-[11px] text-zinc-400 dark:text-zinc-500">
              绿点标记当前使用的供应商；在主页「开始分析」旁的下拉中切换。
            </p>
          </div>

          {editing && (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                    名称
                  </label>
                  <input
                    value={editing.name}
                    onChange={(e) => patchProvider({ name: e.target.value })}
                    placeholder="如 DeepSeek"
                    className={inputCls}
                    spellCheck={false}
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                    接口类型
                  </label>
                  <div className="relative">
                    <select
                      value={editing.protocol || 'auto'}
                      onChange={(e) => patchProvider({ protocol: e.target.value })}
                      className={`${inputCls} appearance-none pr-9`}
                    >
                      {PROTOCOLS.map((p) => (
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
              </div>

              <div>
                <label className="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Base URL
                </label>
                <input
                  value={editing.base_url}
                  onChange={(e) => patchProvider({ base_url: e.target.value })}
                  placeholder="https://api.deepseek.com/v1"
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
                  value={editing.api_key}
                  onChange={(e) => patchProvider({ api_key: e.target.value })}
                  placeholder={editing.api_key ? `${maskKey(editing.api_key)}（已保存）` : 'sk-...'}
                  className={inputCls}
                  autoComplete="off"
                  spellCheck={false}
                />
              </div>

              {/* 已添加模型 */}
              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
                    模型
                  </span>
                  <button
                    onClick={() => void handleFetchModels()}
                    disabled={fetching || saving}
                    className="inline-flex h-7 items-center gap-1 rounded-lg border border-zinc-200 px-2 text-xs text-zinc-600 transition-colors hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
                  >
                    {fetching ? (
                      <span className="h-3 w-3 animate-spin rounded-full border-2 border-zinc-400 border-t-transparent" />
                    ) : (
                      <RefreshCw className="h-3 w-3" aria-hidden="true" />
                    )}
                    {fetching ? '获取中…' : '获取模型'}
                  </button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {editing.models.map((m) => (
                    <span
                      key={m}
                      className={`inline-flex h-7 items-center gap-1 rounded-full border px-2.5 text-xs ${
                        draft.active_provider_id === editing.id && draft.active_model === m
                          ? 'border-emerald-500/50 bg-emerald-50 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
                          : 'border-zinc-200 text-zinc-600 dark:border-zinc-700 dark:text-zinc-300'
                      }`}
                    >
                      {m}
                      <button
                        onClick={() => removeModel(m)}
                        className="text-zinc-400 transition-colors hover:text-rose-500"
                        aria-label={`删除模型 ${m}`}
                      >
                        <X className="h-3 w-3" aria-hidden="true" />
                      </button>
                    </span>
                  ))}
                  {editing.models.length === 0 && (
                    <span className="text-xs text-zinc-400 dark:text-zinc-500">
                      暂无模型，可手动添加或点「获取模型」后点选
                    </span>
                  )}
                </div>
                <div className="mt-2 flex gap-2">
                  <input
                    value={newModel}
                    onChange={(e) => setNewModel(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        addModel(newModel)
                        setNewModel('')
                      }
                    }}
                    placeholder="手动输入模型 ID，如 deepseek-chat"
                    className={`${inputCls} h-9 py-0 flex-1`}
                    spellCheck={false}
                  />
                  <button
                    onClick={() => {
                      addModel(newModel)
                      setNewModel('')
                    }}
                    disabled={!newModel.trim()}
                    className="inline-flex h-9 shrink-0 items-center gap-1 rounded-lg border border-zinc-200 px-3 text-xs text-zinc-600 transition-colors hover:bg-zinc-50 disabled:opacity-40 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
                  >
                    <Plus className="h-3.5 w-3.5" aria-hidden="true" />
                    添加
                  </button>
                </div>
              </div>

              {/* 获取结果待选区：点哪个加哪个，已添加的置灰 */}
              {available.length > 0 && (
                <div className="rounded-lg border border-dashed border-zinc-300 p-3 dark:border-zinc-700">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
                      可用模型（点击添加，已添加的置灰）
                    </span>
                    <button
                      onClick={() => setAvailable([])}
                      className="text-zinc-400 transition-colors hover:text-zinc-600 dark:hover:text-zinc-200"
                      aria-label="收起可用模型"
                    >
                      <X className="h-3.5 w-3.5" aria-hidden="true" />
                    </button>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {available.map((m) => {
                      const added = editing.models.includes(m)
                      return (
                        <button
                          key={m}
                          onClick={() => addModel(m)}
                          disabled={added}
                          className={`inline-flex h-7 items-center gap-1 rounded-full border px-2.5 text-xs transition-colors ${
                            added
                              ? 'cursor-default border-emerald-500/40 bg-emerald-50 text-emerald-600 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400'
                              : 'border-zinc-300 text-zinc-700 hover:border-zinc-500 hover:bg-zinc-50 dark:border-zinc-600 dark:text-zinc-200 dark:hover:bg-zinc-800'
                          }`}
                        >
                          {added && <Check className="h-3 w-3" aria-hidden="true" />}
                          {m}
                        </button>
                      )
                    })}
                  </div>
                </div>
              )}

              <button
                onClick={removeProvider}
                disabled={saving}
                className="inline-flex items-center gap-1.5 text-xs text-rose-500 transition-colors hover:text-rose-600 disabled:opacity-50"
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                删除该供应商
              </button>
            </>
          )}

          {localError && (
            <div className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2.5 text-sm text-rose-700 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-300">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              {localError}
            </div>
          )}
          {error && !localError && (
            <div className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2.5 text-sm text-rose-700 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-300">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              {error}
            </div>
          )}
          {notice && (
            <div className="flex items-start gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-sm text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              {notice}
            </div>
          )}
        </div>

        <div className="flex shrink-0 items-center justify-between gap-3 border-t border-zinc-100 px-6 py-4 dark:border-zinc-800">
          <button
            onClick={() => void handleTest()}
            disabled={testing || saving || !editing}
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
