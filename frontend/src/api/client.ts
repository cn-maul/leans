import type {
  AISettings,
  AnalysisResult,
  HistoryItem,
  Stats,
  Subject,
} from '../types/analysis'

const BASE_URL = '/api'

// ApiError carries the server-provided message.
export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let resp: Response
  try {
    resp = await fetch(`${BASE_URL}${path}`, {
      headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
      ...init,
    })
  } catch (e) {
    // 用户主动取消时保留 AbortError，供调用方识别。
    if (e instanceof DOMException && e.name === 'AbortError') throw e
    throw new ApiError('网络请求失败，请检查服务是否在运行', 0)
  }

  if (!resp.ok) {
    let message = `请求失败（${resp.status}）`
    try {
      const body = await resp.json()
      if (body?.error) message = body.error
    } catch {
      /* keep fallback message */
    }
    throw new ApiError(message, resp.status)
  }
  return resp.json() as Promise<T>
}

export function fetchSubjects(): Promise<Subject[]> {
  return request<Subject[]>('/subjects')
}

export function fetchStats(): Promise<Stats> {
  return request<Stats>('/stats')
}

export function analyzeQuestion(
  subject: string,
  content: string,
  signal?: AbortSignal,
): Promise<AnalysisResult> {
  return request<AnalysisResult>('/analyze', {
    method: 'POST',
    body: JSON.stringify({ subject, content }),
    signal,
  })
}

export interface StreamHandlers {
  onModel?: (model: string) => void
  onDelta?: (text: string) => void
  onStatus?: (message: string) => void
}

// analyzeQuestionStream 调用 SSE 流式分析端点，delta 事件实时回调，
// 最终以 done 事件中的规范化结果 resolve。signal 取消时以 AbortError 拒绝。
// 非 SSE 响应（旧后端）抛出 404 供调用方回退。
export async function analyzeQuestionStream(
  subject: string,
  content: string,
  handlers: StreamHandlers,
  signal?: AbortSignal,
): Promise<AnalysisResult> {
  let resp: Response
  try {
    resp = await fetch(`${BASE_URL}/analyze/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ subject, content }),
      signal,
    })
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') throw e
    throw new ApiError('网络请求失败，请检查服务是否在运行', 0)
  }

  if (!resp.ok) {
    throw new ApiError(`请求失败（${resp.status}）`, resp.status)
  }
  // 旧版后端没有该端点时 NoRoute 会返回 HTML，据此回退到非流式接口。
  if (!(resp.headers.get('Content-Type') || '').includes('text/event-stream') || !resp.body) {
    throw new ApiError('服务端不支持流式响应', 404)
  }

  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let final: AnalysisResult | null = null
  let errorMessage = ''

  const handleFrame = (frame: string) => {
    let event = 'message'
    let data = ''
    for (const line of frame.split('\n')) {
      if (line.startsWith('event:')) event = line.slice(6).trim()
      else if (line.startsWith('data:')) data += line.slice(5).trim()
    }
    if (!data) return
    let payload: Record<string, unknown>
    try {
      payload = JSON.parse(data)
    } catch {
      return
    }
    if (event === 'delta') handlers.onDelta?.(String(payload.t ?? ''))
    else if (event === 'status') handlers.onStatus?.(String(payload.message ?? ''))
    else if (event === 'model') handlers.onModel?.(String(payload.model ?? ''))
    else if (event === 'done') final = payload as unknown as AnalysisResult
    else if (event === 'error') errorMessage = String(payload.error ?? '分析失败')
  }

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    let idx: number
    while ((idx = buffer.indexOf('\n\n')) >= 0) {
      handleFrame(buffer.slice(0, idx))
      buffer = buffer.slice(idx + 2)
    }
  }
  // 处理结尾未跟空行的最后一帧。
  if (buffer.trim()) handleFrame(buffer)

  if (errorMessage) throw new ApiError(errorMessage, 500)
  if (!final) throw new ApiError('流式响应中断，请重试', 0)
  return final
}

export function fetchSettings(): Promise<AISettings> {
  return request<AISettings>('/settings')
}

export function saveSettings(settings: AISettings): Promise<AISettings> {
  return request<AISettings>('/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  })
}

// setActiveSelection 切换主界面二级下拉的当前供应商与模型；model 传空
// 字符串表示由后端自动选中该供应商下的有效模型。
export function setActiveSelection(providerId: string, model: string): Promise<AISettings> {
  return request<AISettings>('/settings/active', {
    method: 'POST',
    body: JSON.stringify({ provider_id: providerId, model }),
  })
}

// fetchProviderModels 用表单凭据在线获取模型列表（服务端 GET /models）。
export function fetchProviderModels(
  baseUrl: string,
  apiKey: string,
  protocol: string,
): Promise<{ models: string[] }> {
  return request<{ models: string[] }>('/settings/models', {
    method: 'POST',
    body: JSON.stringify({ base_url: baseUrl, api_key: apiKey, protocol }),
  })
}

// testConnection 测试连接：传入 provider 时测试表单配置（可不保存），
// 否则测试当前激活的供应商。
export function testConnection(
  provider?: { name?: string; base_url: string; api_key: string; protocol?: string },
  model?: string,
): Promise<{ ok: boolean }> {
  const body = provider ? { provider, model } : {}
  return request<{ ok: boolean }>('/settings/test', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function fetchHistory(): Promise<HistoryItem[]> {
  return request<HistoryItem[]>('/history')
}

export function fetchHistoryItem(id: number): Promise<HistoryItem> {
  return request<HistoryItem>(`/history/${id}`)
}

export function clearHistory(): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>('/history', { method: 'DELETE' })
}
