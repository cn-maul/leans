import { useCallback, useRef, useState } from 'react'
import * as api from '../api/client'
import type { AnalysisResult, PartialAnalysis } from '../types/analysis'

export type AnalysisStage = 'idle' | 'running' | 'done'

// ---- 流式增量解析：从累积的 JSON 文本中提取已完成的字段 ----

// 提取单个字符串字段的当前值（值未闭合时返回已输出的部分）。
function matchString(src: string, key: string): string {
  const re = new RegExp(`"${key}"\\s*:\\s*"((?:[^"\\\\]|\\\\.)*)`)
  const m = src.match(re)
  if (!m) return ''
  try {
    return JSON.parse(`"${m[1]}"`) as string
  } catch {
    return m[1]
  }
}

// 提取 rules 数组中已完成的 name/section/usage（按下标配对）。
function matchRules(src: string): PartialAnalysis['rules'] {
  const grab = (key: string) => {
    const re = new RegExp(`"${key}"\\s*:\\s*"((?:[^"\\\\]|\\\\.)*)`, 'g')
    const out: string[] = []
    let m: RegExpExecArray | null
    while ((m = re.exec(src)) !== null) {
      try {
        out.push(JSON.parse(`"${m[1]}"`) as string)
      } catch {
        out.push(m[1])
      }
    }
    return out
  }
  const names = grab('name')
  const sections = grab('section')
  const usages = grab('usage')
  const rules: PartialAnalysis['rules'] = []
  for (let i = 0; i < names.length; i++) {
    rules.push({ name: names[i], section: sections[i] ?? '', usage: usages[i] ?? '' })
  }
  return rules
}

function parsePartial(raw: string): PartialAnalysis {
  return {
    category: matchString(raw, 'category'),
    subCategory: matchString(raw, 'sub_category'),
    rules: matchRules(raw),
    answer: matchString(raw, 'answer'),
    annotation: matchString(raw, 'annotation'),
  }
}

export function useAnalysis() {
  const [result, setResult] = useState<AnalysisResult | null>(null)
  const [stage, setStage] = useState<AnalysisStage>('idle')
  const [error, setError] = useState('')
  const [partial, setPartial] = useState<PartialAnalysis | null>(null)
  const [liveModel, setLiveModel] = useState('')
  const [liveFirstTokenMS, setLiveFirstTokenMS] = useState(0)
  const firstTokenAtRef = useRef(0)
  const abortRef = useRef<AbortController | null>(null)

  const isAbort = (e: unknown) =>
    e instanceof DOMException && e.name === 'AbortError'

  const run = useCallback(async (subject: string, question: string) => {
    // 新的一次运行会替换旧的 controller；旧请求若仍在飞行中先中止。
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    setStage('running')
    setError('')
    setPartial(null)
    setLiveModel('')
    setLiveFirstTokenMS(0)
    firstTokenAtRef.current = 0
    const startedAt = Date.now()
    const acc = { text: '' }

    try {
      const res = await api.analyzeQuestionStream(
        subject,
        question,
        {
          onModel: (m) => setLiveModel(m),
          onDelta: (text) => {
            if (firstTokenAtRef.current === 0) {
              firstTokenAtRef.current = Date.now()
              setLiveFirstTokenMS(Date.now() - startedAt)
            }
            acc.text += text
            setPartial(parsePartial(acc.text))
          },
          onStatus: () => {
            /* 状态提示（解析失败重试）在注释卡内以阶段文案体现 */
          },
        },
        controller.signal,
      )
      setResult(res)
      setStage('done')
      return res
    } catch (e) {
      // 旧版后端无流式端点时回退非流式接口。
      if (e instanceof api.ApiError && e.status === 404) {
        try {
          const res = await api.analyzeQuestion(subject, question, controller.signal)
          setResult(res)
          setStage('done')
          return res
        } catch (e2) {
          if (isAbort(e2)) return null
          setError(e2 instanceof Error ? e2.message : '分析失败')
          setStage('idle')
          return null
        }
      }
      if (isAbort(e)) {
        // 用户中止：静默回到初始态，已收到的部分结果一并丢弃。
        setPartial(null)
        setStage('idle')
        return null
      }
      setError(e instanceof Error ? e.message : '分析失败')
      setStage('idle')
      return null
    }
  }, [])

  // cancel 中止进行中的分析；后端会随连接断开取消 AI 调用。
  const cancel = useCallback(() => {
    abortRef.current?.abort()
  }, [])

  const reset = useCallback(() => {
    setResult(null)
    setStage('idle')
    setError('')
    setPartial(null)
    setLiveModel('')
    setLiveFirstTokenMS(0)
  }, [])

  return { result, stage, error, run, cancel, reset, setResult, partial, liveModel, liveFirstTokenMS }
}
