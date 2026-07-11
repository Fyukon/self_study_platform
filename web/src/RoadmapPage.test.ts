import { describe, expect, it } from 'vitest'
import { directionIDFromReference } from './RoadmapPage'
import { renderToStaticMarkup } from 'react-dom/server'
import { createElement } from 'react'
import { Icon } from './icons'
import type { Direction } from './types'

const directions: Direction[] = [{
  id: 7,
  title: 'Backend Go',
  description: '',
  icon: 'server',
  position: 0,
  is_archived: false,
  created_at: '',
  updated_at: '',
}]

describe('roadmap direction links', () => {
  it('resolves numeric and readable URL references', () => {
    expect(directionIDFromReference(directions, '7')).toBe(7)
    expect(directionIDFromReference(directions, 'backend-go')).toBe(7)
    expect(directionIDFromReference(directions, 'Backend%20Go')).toBe(7)
    expect(directionIDFromReference(directions, 'missing')).toBeNull()
  })

  it('renders direction icons as inline SVG instead of the seed value', () => {
    const html = renderToStaticMarkup(createElement(Icon, { name: 'server' }))
    expect(html).toContain('<svg')
    expect(html).not.toContain('>server<')
  })
})
