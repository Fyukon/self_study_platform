import { useState } from 'react'
import type { RoadmapNode, RoadmapStatus } from './types'
import { Icon } from './icons'

export const statusLabels: Record<RoadmapStatus, string> = {
  not_started: 'Не начато',
  learning: 'Изучаю',
  practicing: 'Практикую',
  understood: 'Понимаю',
}

export interface Progress {
  total: number
  understood: number
  percent: number
}

export function calculateProgress(nodes: RoadmapNode[]): Progress {
  let total = 0
  let understood = 0

  const visit = (items: RoadmapNode[]) => {
    for (const node of items) {
      if (node.node_type !== 'section') {
        total++
        if (node.status === 'understood') understood++
      }
      visit(node.children ?? [])
    }
  }

  visit(nodes)
  return { total, understood, percent: total ? Math.round((understood / total) * 100) : 0 }
}

interface RoadmapTreeProps {
  nodes: RoadmapNode[]
  selectedId: number | null
  onSelect: (node: RoadmapNode) => void
}

export function RoadmapTree({ nodes, selectedId, onSelect }: RoadmapTreeProps) {
  return (
    <ul className="roadmap-tree" role="tree" aria-label="Темы roadmap">
      {nodes.map((node) => (
        <TreeItem key={node.id} node={node} depth={0} selectedId={selectedId} onSelect={onSelect} />
      ))}
    </ul>
  )
}

interface TreeItemProps extends Omit<RoadmapTreeProps, 'nodes'> {
  node: RoadmapNode
  depth: number
}

function TreeItem({ node, depth, selectedId, onSelect }: TreeItemProps) {
  const children = node.children ?? []
  const hasChildren = children.length > 0
  const [expanded, setExpanded] = useState(depth === 0)

  return (
    <li
      className={`tree-item ${node.node_type === 'section' ? 'is-section' : ''}`}
      role="treeitem"
      aria-expanded={hasChildren ? expanded : undefined}
    >
      <div className={`tree-row ${selectedId === node.id ? 'selected' : ''}`}>
        {hasChildren ? (
          <button
            className="tree-expander"
            type="button"
            aria-label={expanded ? `Свернуть «${node.title}»` : `Развернуть «${node.title}»`}
            onClick={() => setExpanded(!expanded)}
          >
            <Icon name={expanded ? 'chevron-down' : 'chevron-right'} size={16} />
          </button>
        ) : (
          <span className="tree-expander-spacer" aria-hidden="true" />
        )}

        <button className="tree-main" type="button" onClick={() => onSelect(node)}>
          <span className="tree-title">{node.title}</span>
          {node.node_type !== 'section' && (
            <span className="tree-meta">
              <span className={`status-badge status-${node.status}`}>{statusLabels[node.status]}</span>
              {node.confidence > 0 && <span>Уверенность {node.confidence}/5</span>}
              {node.needs_review && <span className="review-badge">Повторить</span>}
            </span>
          )}
          {node.next_action && <span className="tree-action">Дальше: {node.next_action}</span>}
        </button>
      </div>

      {hasChildren && expanded && (
        <ul role="group">
          {children.map((child) => (
            <TreeItem
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              onSelect={onSelect}
            />
          ))}
        </ul>
      )}
    </li>
  )
}
