export interface Subject {
  id: string
  name: string
}

// AIProvider 是一个可切换的 AI 服务接入点；models 可手动维护或在线获取。
// protocol 取值与后端 model 包常量一致：auto / openai-chat / openai-responses / anthropic。
export interface AIProvider {
  id: string
  name: string
  base_url: string
  api_key: string
  protocol: string
  models: string[]
}

export interface AISettings {
  active_provider_id: string
  active_model: string
  providers: AIProvider[]
}

export interface HistoryItem {
  id: number
  subject: string
  question: string
  category: string
  result?: string
  model?: string
  tokens: number
  elapsed_ms: number
  first_token_ms?: number
  created_at: string
}

// 统计页数据，对应 GET /api/stats。
export interface Stats {
  total_questions: number
  total_tokens: number
  avg_tokens: number
  avg_first_token_ms: number
  total_elapsed_ms: number
}

export type HighlightColor = 'green' | 'red' | 'blue' | 'yellow' | 'ink'
export type HighlightModule = 'category' | 'rule' | 'annotation' | 'error' | 'info'

export interface Highlight {
  text: string
  type: string
  color?: HighlightColor | string
  module?: HighlightModule | string
  location: string
  explanation: string
}

export interface Rule {
  name: string
  section: string
  // application 是新版的"技巧在这道题里怎么用"；usage 为旧数据兜底。
  application?: string
  usage?: string
  example?: string
  marks?: Highlight[]
}

export interface TypeJudgment {
  category: string
  sub_category: string
  basis?: Highlight[]
}

export interface TechniqueJudgment {
  rules?: Rule[]
}

export interface AnalysisMeta {
  elapsed_ms: number
  first_token_ms?: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  model?: string
}

// 流式过程中的增量解析结果：字段随 AI 输出逐步填充。
export interface PartialAnalysis {
  category: string
  subCategory: string
  rules: Array<{ name: string; section: string; usage: string }>
  answer: string
  annotation: string
}

export interface TechniqueInfo {
  name: string
  section: string
  description: string
}

export interface ApplicableRule {
  rule_name: string
  section: string
  usage: string
  example: string
}

export interface AnalysisResult {
  type_judgment?: TypeJudgment
  technique_judgment?: TechniqueJudgment
  category: string
  sub_category: string
  answer?: string
  basis?: Highlight[]
  rules?: Rule[]
  annotation?: string
  highlights: Highlight[]
  meta: AnalysisMeta
  // 兼容旧数据
  techniques?: TechniqueInfo[]
  applicable?: ApplicableRule[]
}

// module -> 颜色，左栏标注与右栏模块使用同一映射，保证颜色一致。
// annotation 用墨色（ink），避免蓝紫 AI 风。
export const MODULE_COLOR: Record<string, HighlightColor> = {
  category: 'blue',
  rule: 'green',
  annotation: 'ink',
  error: 'red',
  info: 'yellow',
}

export const MODULE_LABEL: Record<string, string> = {
  category: '题型判断',
  rule: '适用规则',
  annotation: '答案/思路',
  error: '错误/转折',
  info: '关键信息',
}

// 旧数据的 color 字段兜底映射（旧版 annotation=紫 → 墨色）。
export const LEGACY_COLOR: Record<string, HighlightColor> = {
  purple: 'ink',
}

export type Theme = 'light' | 'dark'
