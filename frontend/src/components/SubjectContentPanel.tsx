import { useState } from 'react'
import type { Section, SubjectContent } from '../types/analysis'

interface Props {
  content: SubjectContent | null
  loading?: boolean
}

// SubjectContentPanel 在抽屉内展示讲义章节树，点击章节在下方查看正文。
export default function SubjectContentPanel({ content, loading }: Props) {
  const [selected, setSelected] = useState<Section | null>(null)

  if (loading) {
    return (
      <div className="space-y-2 animate-pulse">
        <div className="h-4 bg-gray-100 dark:bg-gray-800 rounded" />
        <div className="h-4 bg-gray-100 dark:bg-gray-800 rounded w-3/4" />
        <div className="h-4 bg-gray-100 dark:bg-gray-800 rounded w-1/2" />
      </div>
    )
  }

  if (!content) {
    return (
      <div className="text-sm text-gray-400 dark:text-gray-500 text-center py-8">
        请先在顶栏选择一个科目
      </div>
    )
  }

  return (
    <div>
      {content.summary && (
        <details className="mb-2 group">
          <summary className="cursor-pointer select-none text-xs font-medium text-primary-600 dark:text-primary-400 hover:underline">
            开篇概述
          </summary>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400 leading-relaxed whitespace-pre-wrap">
            {content.summary}
          </p>
        </details>
      )}

      <SectionTree
        sections={content.tree}
        selectedId={selected?.id ?? null}
        onSelect={setSelected}
      />

      {selected && selected.content && (
        <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-800">
          <div className="text-xs font-semibold text-gray-700 dark:text-gray-200 mb-1.5">
            {selected.title}
          </div>
          <p className="text-xs text-gray-600 dark:text-gray-300 leading-relaxed whitespace-pre-wrap max-h-80 overflow-y-auto">
            {selected.content}
          </p>
        </div>
      )}
    </div>
  )
}

function SectionTree({
  sections,
  selectedId,
  onSelect,
}: {
  sections: Section[]
  selectedId: string | null
  onSelect: (s: Section) => void
}) {
  const [expanded, setExpanded] = useState<Set<string>>(
    () => new Set(sections.map((s) => s.id)),
  )

  const toggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  if (sections.length === 0) {
    return <div className="text-xs text-gray-400 dark:text-gray-500 py-4 text-center">本讲义无章节</div>
  }

  return (
    <div className="space-y-0.5">
      {sections.map((s) => (
        <TreeNode
          key={s.id}
          section={s}
          depth={0}
          expanded={expanded}
          onToggle={toggle}
          selectedId={selectedId}
          onSelect={onSelect}
        />
      ))}
    </div>
  )
}

function TreeNode({
  section,
  depth,
  expanded,
  onToggle,
  selectedId,
  onSelect,
}: {
  section: Section
  depth: number
  expanded: Set<string>
  onToggle: (id: string) => void
  selectedId: string | null
  onSelect: (s: Section) => void
}) {
  const hasChildren = !!section.children?.length
  const isOpen = expanded.has(section.id)
  const isSelected = selectedId === section.id

  return (
    <div>
      <div
        className={`flex items-start gap-1.5 px-2 py-1.5 rounded-md text-sm transition-colors ${
          isSelected
            ? 'bg-primary-50 dark:bg-primary-900/40'
            : 'hover:bg-gray-50 dark:hover:bg-gray-800'
        }`}
        style={{ paddingLeft: depth * 14 + 8 }}
      >
        {hasChildren ? (
          <button
            onClick={() => onToggle(section.id)}
            className="mt-0.5 text-gray-400 dark:text-gray-500 text-xs transition-transform hover:text-gray-600 dark:hover:text-gray-300"
            aria-label={isOpen ? '收起' : '展开'}
          >
            <span className={`inline-block transition-transform ${isOpen ? 'rotate-90' : ''}`}>▶</span>
          </button>
        ) : (
          <span className="mt-0.5 text-gray-300 dark:text-gray-600 text-xs">•</span>
        )}
        <button
          onClick={() => onSelect(section)}
          className={`flex-1 text-left ${
            hasChildren
              ? 'font-medium text-gray-700 dark:text-gray-200'
              : 'text-gray-600 dark:text-gray-300'
          }`}
        >
          {section.title}
        </button>
      </div>

      {isOpen && hasChildren && (
        <div className="ml-1 border-l border-gray-100 dark:border-gray-800 pl-2">
          {section.children!.map((c) => (
            <TreeNode
              key={c.id}
              section={c}
              depth={depth + 1}
              expanded={expanded}
              onToggle={onToggle}
              selectedId={selectedId}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  )
}
