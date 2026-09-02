export interface Subject {
  id: string
  name: string
}

export interface AISettings {
  provider: string
  api_key: string
  base_url: string
  model: string
}

export interface HistoryItem {
  id: number
  subject: string
  question: string
  category: string
  result?: string
  created_at: string
}

export interface Section {
  id: string
  title: string
  level: number
  content: string
  children?: Section[]
}

export interface SubjectContent {
  id: string
  name: string
  summary: string
  tree: Section[]
}

export type HighlightColor = 'green' | 'red' | 'blue' | 'yellow' | 'purple'
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
  usage: string
  example: string
  marks?: Highlight[]
}

export interface AnalysisMeta {
  elapsed_ms: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
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
export const MODULE_COLOR: Record<string, HighlightColor> = {
  category: 'blue',
  rule: 'green',
  annotation: 'purple',
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

export type Theme = 'light' | 'dark'
