// 共享标注引擎：把一组 Highlight 容错匹配到一段原文上，产出着色区间。
// 客观题（题干/选项）与主观题（给定材料/我的作答）两套标注视图都复用它，
// 差异只在"如何切块 + 把 highlight 归到哪块"，匹配算法完全一致。
import type { Highlight } from '../types/analysis'

export interface Block {
  key: string
  label: string
  text: string
}

export interface MarkRange {
  start: number
  end: number
  cls: string
  tip: string
}

// ---- 颜色/模块映射（左右两栏统一，保证同模块同色）----

// module → 标记类。色值定义在 index.css，浅/暗两套各一份。
export const MODULE_MARK: Record<string, string> = {
  category: 'mark mark-category',
  rule: 'mark mark-rule',
  annotation: 'mark mark-answer',
  error: 'mark mark-error',
  info: 'mark mark-info',
}

// 旧数据只有 color 字段时的兜底（紫 → 墨色）。
export const COLOR_MARK: Record<string, string> = {
  blue: 'mark mark-category',
  green: 'mark mark-rule',
  ink: 'mark mark-answer',
  purple: 'mark mark-answer',
  red: 'mark mark-error',
  yellow: 'mark mark-info',
}

export const MODULE_DOT: Record<string, string> = {
  category: 'dot-category',
  rule: 'dot-rule',
  annotation: 'dot-answer',
  error: 'dot-error',
  info: 'dot-info',
}

export function markClassOf(h: Highlight): string {
  return MODULE_MARK[h.module || ''] || COLOR_MARK[h.color || ''] || MODULE_MARK.category
}

// ---- 容错匹配 ----

// Normalize a single char: full-width -> half-width, lowercase.
function normChar(ch: string): string {
  const code = ch.codePointAt(0) ?? 0
  if (code >= 0xff01 && code <= 0xff5e) {
    return String.fromCodePoint(code - 0xfee0).toLowerCase()
  }
  if (code === 0x3000) return ' '
  return ch.toLowerCase()
}

// Returns the normalized text plus a map from normalized index -> original index.
function normalizeWithMap(text: string): { norm: string; map: number[] } {
  const out: string[] = []
  const map: number[] = []
  for (let i = 0; i < text.length; i++) {
    out.push(normChar(text[i]))
    map.push(i)
  }
  return { norm: out.join(''), map }
}

// Compact form: strip whitespace + punctuation from normalized text, keeping a
// map from compact index -> normalized index.
function compactForm(text: string): { text: string; map: number[] } {
  const out: string[] = []
  const map: number[] = []
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (/\s|[　-〿！-￯，。、；：？！,.!?;:'"“”‘’（）()【】[]<>《》—…·-]/.test(ch)) {
      continue
    }
    out.push(ch)
    map.push(i)
  }
  return { text: out.join(''), map }
}

// locate 在一段原文里找出某条 highlight 的字符区间；找不到返回 null。
// 两级容错：先按规范化子串，再按去标点/空白的紧凑子串。
export function locate(text: string, phrase: string): { start: number; end: number } | null {
  if (!phrase) return null
  const { norm, map } = normalizeWithMap(text)
  const needleNorm = normalizeWithMap(phrase).norm.trim()
  if (!needleNorm) return null

  let idx = norm.indexOf(needleNorm)
  if (idx !== -1) {
    return { start: map[idx], end: map[idx + needleNorm.length - 1] }
  }

  const needleCompact = compactForm(needleNorm).text
  if (needleCompact) {
    const compact = compactForm(norm)
    const ci = compact.text.indexOf(needleCompact)
    if (ci !== -1) {
      const startN = compact.map[ci]
      const endN = compact.map[ci + needleCompact.length - 1]
      return { start: map[startN], end: map[endN] }
    }
  }
  return null
}

// findMarkRanges 对一段文本匹配多条 highlight，合并重叠区间并按位置排序。
export function findMarkRanges(text: string, hs: Highlight[]): MarkRange[] {
  const ranges: MarkRange[] = []
  for (const h of hs) {
    const loc = locate(text, h.text)
    if (!loc) continue
    ranges.push({
      start: loc.start,
      end: loc.end,
      cls: markClassOf(h),
      tip: h.type + (h.explanation ? '：' + h.explanation : ''),
    })
  }
  if (ranges.length === 0) return []

  ranges.sort((a, b) => a.start - b.start)
  const merged: MarkRange[] = []
  for (const r of ranges) {
    const last = merged[merged.length - 1]
    if (last && r.start <= last.end) {
      if (r.end > last.end) last.end = r.end
    } else {
      merged.push({ ...r })
    }
  }
  return merged
}

// routeByLocation 按 highlight.location 归块（客观题：题干/选项A–H）。
export function normalizeLocKey(loc: string): string {
  const s = (loc || '').trim()
  const m = s.match(/[A-H]/i)
  if (m) return m[0].toUpperCase()
  return 'stem'
}

export function groupByLocation(highlights: Highlight[]): Record<string, Highlight[]> {
  const groups: Record<string, Highlight[]> = {}
  for (const h of highlights) {
    const key = normalizeLocKey(h.location)
    ;(groups[key] ||= []).push(h)
  }
  return groups
}

// routeByText 忽略 AI 给的 location，把每条 highlight 落到它文本真正命中的块。
// 材料多段落时更稳：AI 常报错段落序号，但 highlight.text 是从原文摘的。
// 一条命中多块时取首个；一条都命中不到则归入 fallbackKey（默认第一块）。
export function routeByText(
  blocks: Block[],
  highlights: Highlight[],
  fallbackKey?: string,
): Record<string, Highlight[]> {
  const groups: Record<string, Highlight[]> = {}
  const fb = fallbackKey ?? blocks[0]?.key ?? 'stem'
  for (const h of highlights) {
    let key = fb
    for (const b of blocks) {
      if (locate(b.text, h.text)) {
        key = b.key
        break
      }
    }
    ;(groups[key] ||= []).push(h)
  }
  return groups
}
