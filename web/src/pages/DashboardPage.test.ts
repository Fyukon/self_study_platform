import { describe, expect, it } from 'vitest'
import { summarizeToday, type RoadmapSnapshot } from './DashboardPage'
import type { Direction, RoadmapNode } from '../shared/types'

const direction: Direction = {
  id: 1,
  title: 'Backend Go',
  description: '',
  icon: 'server',
  position: 0,
  is_archived: false,
  created_at: '',
  updated_at: '',
}

let nextID = 1

function node(overrides: Partial<RoadmapNode>): RoadmapNode {
  return {
    id: nextID++,
    direction_id: direction.id,
    parent_id: 100,
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

describe('today dashboard summary', () => {
  it('finds review, overdue, available and blocked topics across the tree', () => {
    const review = node({
      id: 1,
      title: 'HTTP',
      status: 'learning',
      needs_review: true,
      last_reviewed_at: '2026-07-06T10:00:00Z',
      estimated_hours: 2,
    })
    const overdue = node({
      id: 2,
      title: 'TLS',
      target_date: '2026-07-10',
      dependencies: [{ id: 1, node_id: 2, depends_on_node_id: review.id, created_at: '' }],
    })
    const available = node({
      id: 3,
      title: 'SQL',
      next_action: 'Решить 3 задачи',
      position: 1,
    })
    const completed = node({
      id: 4,
      title: 'DNS',
      status: 'understood',
      last_reviewed_at: '2026-07-07T10:00:00Z',
      estimated_hours: 1,
    })
    const snapshots: RoadmapSnapshot[] = [{
      direction,
      nodes: [node({ id: 100, node_type: 'section', children: [review, overdue, available, completed] })],
    }]

    const summary = summarizeToday(snapshots, new Date('2026-07-11T12:00:00Z'))

    expect(summary.review.map((item) => item.node.title)).toEqual(['HTTP'])
    expect(summary.overdue.map((item) => item.node.title)).toEqual(['TLS'])
    expect(summary.actions.map((item) => item.node.title)).toEqual(['HTTP', 'SQL'])
    expect(summary.blocked[0].blockers.map((item) => item.title)).toEqual(['HTTP'])
    expect(summary.week).toMatchObject({ reviewed: 2, completed: 1, hours: 3 })
    expect(summary.week.days.filter((day) => day.count)).toHaveLength(2)
  })
})
