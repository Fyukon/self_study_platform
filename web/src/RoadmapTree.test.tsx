import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { calculateProgress, RoadmapTree } from './RoadmapTree'
import type { RoadmapNode } from './types'

let nextId = 1

function node(overrides: Partial<RoadmapNode>): RoadmapNode {
  return {
    id: nextId++,
    direction_id: 1,
    parent_id: null,
    title: 'Тема',
    description: '',
    node_type: 'concept',
    status: 'not_started',
    needs_review: false,
    confidence: 1,
    position: 0,
    next_action: '',
    target_date: null,
    last_reviewed_at: null,
    estimated_hours: null,
    created_at: '',
    updated_at: '',
    dependencies: [],
    ...overrides,
  }
}

describe('roadmap tree', () => {
  it('renders nested topics and their progress markers', () => {
    const child = node({
      title: 'HTTP',
      parent_id: 100,
      status: 'learning',
      confidence: 3,
      needs_review: true,
    })
    const tree = [node({ id: 100, title: 'Основы backend', node_type: 'section', children: [child] })]

    const html = renderToStaticMarkup(<RoadmapTree nodes={tree} selectedId={child.id} onSelect={() => {}} />)

    expect(html).toContain('Основы backend')
    expect(html).toContain('HTTP')
    expect(html).toContain('Изучаю')
    expect(html).toContain('Уверенность 3/5')
    expect(html).toContain('Повторить')
  })

  it('counts understood topics recursively without counting section headings', () => {
    const tree = [
      node({
        node_type: 'section',
        children: [
          node({ status: 'understood' }),
          node({ status: 'learning', children: [node({ status: 'not_started' })] }),
        ],
      }),
    ]

    expect(calculateProgress(tree)).toEqual({ total: 3, understood: 1, percent: 33 })
  })
})
