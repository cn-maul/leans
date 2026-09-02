import type {
  AISettings,
  AnalysisResult,
  HistoryItem,
  Subject,
  SubjectContent,
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
  } catch {
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

export function fetchSubjectContent(id: string): Promise<SubjectContent> {
  return request<SubjectContent>(`/subjects/${encodeURIComponent(id)}`)
}

export function analyzeQuestion(subject: string, content: string): Promise<AnalysisResult> {
  return request<AnalysisResult>('/analyze', {
    method: 'POST',
    body: JSON.stringify({ subject, content }),
  })
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

export function testConnection(): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>('/settings/test', { method: 'POST' })
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
