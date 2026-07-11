import { Link } from 'react-router-dom'
import { Icon } from './icons'

export function Dashboard() {
  return (
    <section className="dashboard page-pad" aria-labelledby="dashboard-title">
      <p className="eyebrow">Обзор</p>
      <h1 id="dashboard-title">Добро пожаловать</h1>
      <div className="empty-card dashboard-empty">
        <div className="empty-symbol"><Icon name="arrow-up-right" size={24} /></div>
        <h2>Начните с карты обучения</h2>
        <p>
          Здесь позже появятся текущие действия и учебная активность. Сейчас доступен Roadmap — основа
          первых двух этапов.
        </p>
        <Link className="button primary" to="/roadmap">Открыть Roadmap</Link>
      </div>
    </section>
  )
}
