import { useEffect, useState } from 'react'
import { NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { CoursesPage } from '../pages/CoursesPage'
import { DashboardPage } from '../pages/DashboardPage'
import { RoadmapPage } from '../pages/RoadmapPage'
import { Icon } from '../shared/components/Icon'

type Theme = 'light' | 'dark'

function initialTheme(): Theme {
  const saved = localStorage.getItem('theme')
  if (saved === 'light' || saved === 'dark') return saved
  return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function App() {
  const [theme, setTheme] = useState<Theme>(initialTheme)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('theme', theme)
  }, [theme])

  return (
    <div className="app-shell">
      <aside className="app-nav">
        <div className="brand">
          <span className="brand-mark"><Icon name="brand" size={20} /></span>
          <span>Learning Roadmap</span>
        </div>

        <nav aria-label="Основная навигация">
          <NavLink to="/" end className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}>
            <Icon name="home" size={18} /> Обзор
          </NavLink>
          <NavLink
            to="/roadmap"
            className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
          >
            <Icon name="roadmap" size={18} /> Roadmap
          </NavLink>
          <NavLink
            to="/courses"
            className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
          >
            <Icon name="book" size={18} /> Курсы
          </NavLink>
          <a className="nav-link" href="/architecture.html">
            <Icon name="compass" size={18} /> Архитектура
          </a>
        </nav>

        <div className="future-nav" aria-label="Будущие разделы">
          <p>Следующие этапы</p>
          <span><Icon name="journal" size={15} /> Журнал</span>
          <span><Icon name="briefcase" size={15} /> Проекты</span>
          <span><Icon name="chart" size={15} /> Аналитика</span>
        </div>

        <button
          className="theme-toggle"
          type="button"
          onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
          aria-label={theme === 'light' ? 'Включить тёмную тему' : 'Включить светлую тему'}
        >
          <Icon name={theme === 'light' ? 'moon' : 'sun'} size={18} />
          {theme === 'light' ? 'Тёмная тема' : 'Светлая тема'}
        </button>
      </aside>

      <main className="app-content">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/roadmap" element={<RoadmapPage />} />
          <Route path="/roadmap/:directionId" element={<RoadmapPage />} />
          <Route path="/courses" element={<CoursesPage />} />
          <Route path="*" element={<Navigate to="/roadmap" replace />} />
        </Routes>
      </main>
    </div>
  )
}
