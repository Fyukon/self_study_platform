import { describe, expect, it } from 'vitest'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { CoursesPage, courseResourceHref } from './CoursesPage'

describe('course resources', () => {
  it('uses a browser URL for remote materials and a file URL for local notes', () => {
    expect(courseResourceHref({ resource_type: 'book', url: 'https://example.com/book', local_path: '' })).toBe('https://example.com/book')
    expect(courseResourceHref({ resource_type: 'local_file', url: '', local_path: '/home/user/notes.md' })).toBe('file:///home/user/notes.md')
  })

  it('renders the course workspace shell before data arrives', () => {
    const html = renderToStaticMarkup(createElement(CoursesPage))
    expect(html).toContain('Учитесь по своему маршруту')
    expect(html).toContain('Импорт из Markdown / JSON')
  })
})
