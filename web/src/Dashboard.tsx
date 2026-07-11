import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from './api'
import { flattenNodes, statusLabels } from './RoadmapTree'
import { Icon, type IconName } from './icons'
import type { Direction, RoadmapNode } from './types'

export interface RoadmapSnapshot {
  direction: Direction
  nodes: RoadmapNode[]
}

export interface TodayItem {
  direction: Direction
  node: RoadmapNode
  blockers: RoadmapNode[]
}

export interface WeekDay {
  key: string
  label: string
  count: number
}

export interface TodaySummary {
  review: TodayItem[]
  overdue: TodayItem[]
  actions: TodayItem[]
  blocked: TodayItem[]
  week: {
    reviewed: number
    completed: number
    hours: number
    days: WeekDay[]
  }
}

type TodayListKind = 'review' | 'overdue' | 'blocked'

const statusPriority: Record<RoadmapNode['status'], number> = {
  learning: 0,
  practicing: 1,
  not_started: 2,
  understood: 3,
}

const listIcons: Record<TodayListKind, IconName> = {
  review: 'journal',
  overdue: 'chart',
  blocked: 'roadmap',
}

const dayLabels = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

function localDateKey(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function startOfWeek(now: Date): Date {
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  const day = start.getDay()
  start.setDate(start.getDate() - (day === 0 ? 6 : day - 1))
  return start
}

function isThisWeek(value: string | null, weekStart: Date, now: Date): boolean {
  if (!value) return false
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) && timestamp >= weekStart.getTime() && timestamp <= now.getTime()
}

function formatDate(value: string | null): string {
  if (!value) return ''
  const date = new Date(`${value}T00:00:00`)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short' }).format(date)
}

export function summarizeToday(snapshots: RoadmapSnapshot[], now = new Date()): TodaySummary {
  const entries = snapshots.flatMap(({ direction, nodes }) =>
    flattenNodes(nodes)
      .filter((node) => node.node_type !== 'section')
      .map((node) => ({ direction, node })),
  )
  const byID = new Map(entries.map((entry) => [entry.node.id, entry.node]))
  const items = entries.map(({ direction, node }) => {
    const blockers = node.dependencies
      .map((dependency) => byID.get(dependency.depends_on_node_id))
      .filter((candidate): candidate is RoadmapNode => candidate !== undefined && candidate.status !== 'understood')
    return { direction, node, blockers }
  })
  const today = localDateKey(now)
  const review = items.filter(({ node }) => node.needs_review)
  const overdue = items.filter(({ node }) => Boolean(node.target_date && node.target_date < today && node.status !== 'understood'))
  const available = items
    .filter(({ node, blockers }) => node.status !== 'understood' && blockers.length === 0)
    .sort((left, right) => {
      const leftDate = left.node.target_date ?? '9999-12-31'
      const rightDate = right.node.target_date ?? '9999-12-31'
      return leftDate.localeCompare(rightDate)
        || statusPriority[left.node.status] - statusPriority[right.node.status]
        || left.node.position - right.node.position
        || left.node.id - right.node.id
    })
  const blocked = items.filter(({ blockers }) => blockers.length > 0)
  const weekStart = startOfWeek(now)
  const reviewedByDay = new Map<string, number>()
  let reviewed = 0
  let completed = 0
  let hours = 0
  for (const { node } of items) {
    if (!isThisWeek(node.last_reviewed_at, weekStart, now)) continue
    reviewed++
    if (node.status === 'understood') completed++
    hours += node.estimated_hours ?? 0
    const key = localDateKey(new Date(node.last_reviewed_at as string))
    reviewedByDay.set(key, (reviewedByDay.get(key) ?? 0) + 1)
  }
  const days = dayLabels.map((label, index) => {
    const date = new Date(weekStart)
    date.setDate(date.getDate() + index)
    const key = localDateKey(date)
    return { key, label, count: reviewedByDay.get(key) ?? 0 }
  })
  return { review, overdue, actions: available.slice(0, 5), blocked, week: { reviewed, completed, hours, days } }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Не удалось загрузить данные для экрана «Сегодня»'
}

function topicHref(item: TodayItem): string {
  return `/roadmap/${item.direction.id}?node=${item.node.id}`
}

export function Dashboard() {
  const [snapshots, setSnapshots] = useState<RoadmapSnapshot[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reload, setReload] = useState(0)
  const summary = useMemo(() => summarizeToday(snapshots), [snapshots])

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setError('')
    api.directions(controller.signal)
      .then((directions) => Promise.all(
        directions
          .filter((direction) => !direction.is_archived)
          .map(async (direction) => ({ direction, nodes: await api.roadmap(direction.id, controller.signal) })),
      ))
      .then(setSnapshots)
      .catch((requestError) => {
        if (!(requestError instanceof DOMException && requestError.name === 'AbortError')) {
          setError(errorMessage(requestError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [reload])

  const todayLabel = new Intl.DateTimeFormat('ru-RU', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  }).format(new Date())

  return (
    <section className="dashboard page-pad today-page" aria-labelledby="dashboard-title">
      <header className="today-header">
        <div>
          <p className="eyebrow">Сегодня</p>
          <h1 id="dashboard-title">Что изучать сейчас?</h1>
          <p className="today-date">{todayLabel}</p>
        </div>
        <Link className="button secondary" to="/roadmap"><Icon name="roadmap" size={16} /> Открыть roadmap</Link>
      </header>

      {loading ? (
        <TodaySkeleton />
      ) : error ? (
        <div className="today-error" role="alert">
          <Icon name="compass" size={24} />
          <strong>Не удалось собрать сегодняшний план</strong>
          <p>{error}</p>
          <button className="button secondary small" type="button" onClick={() => setReload((value) => value + 1)}>Повторить</button>
        </div>
      ) : snapshots.length === 0 ? (
        <div className="empty-card today-empty">
          <div className="empty-symbol"><Icon name="roadmap" size={24} /></div>
          <h2>Сначала добавьте направление</h2>
          <p>Когда появится roadmap, этот экран соберёт конкретные действия на сегодня.</p>
          <Link className="button primary" to="/roadmap">Открыть Roadmap</Link>
        </div>
      ) : (
        <TodayOverview summary={summary} />
      )}
    </section>
  )
}

function TodayOverview({ summary }: { summary: TodaySummary }) {
  return (
    <div className="today-content">
      <div className="today-stat-grid" aria-label="Сводка на сегодня">
        <TodayStat icon="journal" value={summary.review.length} label="повторить" tone="review" />
        <TodayStat icon="chart" value={summary.overdue.length} label="просрочено" tone="overdue" />
        <TodayStat icon="arrow-up-right" value={summary.actions.length} label="доступных действий" tone="actions" />
        <TodayStat icon="roadmap" value={summary.blocked.length} label="заблокировано" tone="blocked" />
      </div>

      <div className="today-grid">
        <section className="today-card today-actions-card" aria-labelledby="today-actions-title">
          <div className="today-card-heading">
            <div>
              <p className="eyebrow">Фокус</p>
              <h2 id="today-actions-title">Следующие действия</h2>
            </div>
            <span className="today-card-count">{summary.actions.length}/5</span>
          </div>
          {summary.actions.length > 0 ? (
            <div className="today-action-list">
              {summary.actions.map((item, index) => (
                <Link className="today-action" key={item.node.id} to={topicHref(item)}>
                  <span className="today-action-number">{String(index + 1).padStart(2, '0')}</span>
                  <span className="today-action-copy">
                    <strong>{item.node.next_action || `Изучить «${item.node.title}»`}</strong>
                    <small>{item.node.title} · {item.direction.title}{item.node.estimated_hours ? ` · ${item.node.estimated_hours} ч` : ''}</small>
                  </span>
                  <Icon name="arrow-up-right" size={16} />
                </Link>
              ))}
            </div>
          ) : (
            <TodayEmpty text="Нет свободных действий. Проверьте блокировки или добавьте новую тему." />
          )}
        </section>

        <WeeklyProgress week={summary.week} />
      </div>

      <div className="today-grid">
        <TodayListSection
          id="today-review-title"
          title="Нужно повторить"
          eyebrow="Память"
          kind="review"
          items={summary.review}
          emptyText="Тем с отметкой «Повторить» нет. Хороший знак."
        />
        <TodayListSection
          id="today-overdue-title"
          title="Просроченные темы"
          eyebrow="Сроки"
          kind="overdue"
          items={summary.overdue}
          emptyText="Просроченных тем нет."
        />
      </div>

      <TodayListSection
        id="today-blocked-title"
        title="Заблокировано зависимостями"
        eyebrow="Следующий уровень"
        kind="blocked"
        items={summary.blocked}
        emptyText="Все доступные зависимости закрыты — можно двигаться дальше."
      />
    </div>
  )
}

function TodayStat({ icon, value, label, tone }: { icon: IconName; value: number; label: string; tone: string }) {
  return (
    <div className={`today-stat today-stat-${tone}`}>
      <span className="today-stat-icon"><Icon name={icon} size={17} /></span>
      <strong>{value}</strong>
      <span>{label}</span>
    </div>
  )
}

function WeeklyProgress({ week }: { week: TodaySummary['week'] }) {
  const maxCount = Math.max(...week.days.map((day) => day.count), 1)
  return (
    <section className="today-card weekly-card" aria-labelledby="weekly-progress-title">
      <div className="today-card-heading">
        <div>
          <p className="eyebrow">Ритм обучения</p>
          <h2 id="weekly-progress-title">Прогресс за неделю</h2>
        </div>
        <span className="week-total"><strong>{week.reviewed}</strong> тем</span>
      </div>
      <div className="week-chart" aria-label={`За неделю отмечено тем: ${week.reviewed}`}>
        {week.days.map((day) => (
          <div className="week-day" key={day.key}>
            <strong>{day.count || ''}</strong>
            <span className="week-bar"><i style={{ height: `${day.count ? Math.max(12, (day.count / maxCount) * 100) : 5}%` }} /></span>
            <small>{day.label}</small>
          </div>
        ))}
      </div>
      <div className="week-footnote">
        <span><strong>{week.completed}</strong> освоено</span>
        <span><strong>{week.hours ? `${week.hours} ч` : '—'}</strong> оценки времени</span>
      </div>
    </section>
  )
}

function TodayListSection({
  id,
  title,
  eyebrow,
  kind,
  items,
  emptyText,
}: {
  id: string
  title: string
  eyebrow: string
  kind: TodayListKind
  items: TodayItem[]
  emptyText: string
}) {
  return (
    <section className="today-card today-list-card" aria-labelledby={id}>
      <div className="today-card-heading">
        <div>
          <p className="eyebrow">{eyebrow}</p>
          <h2 id={id}>{title}</h2>
        </div>
        <span className={`today-list-mark today-list-mark-${kind}`}><Icon name={listIcons[kind]} size={16} /></span>
      </div>
      {items.length > 0 ? (
        <div className="today-topic-list">
          {items.slice(0, 5).map((item) => <TodayTopic key={item.node.id} item={item} kind={kind} />)}
          {items.length > 5 && <p className="today-more">Ещё {items.length - 5}</p>}
        </div>
      ) : (
        <TodayEmpty text={emptyText} />
      )}
    </section>
  )
}

function TodayTopic({ item, kind }: { item: TodayItem; kind: TodayListKind }) {
  const meta = kind === 'blocked'
    ? `Зависит от: ${item.blockers.map((blocker) => blocker.title).join(', ')}`
    : kind === 'overdue'
      ? `Срок был ${formatDate(item.node.target_date)}`
      : `${item.direction.title} · ${item.node.last_reviewed_at ? `отмечено ${formatDate(item.node.last_reviewed_at.slice(0, 10))}` : 'не отмечено'}`
  return (
    <Link className="today-topic" to={topicHref(item)}>
      <span className={`today-topic-icon today-topic-icon-${kind}`}><Icon name={listIcons[kind]} size={16} /></span>
      <span className="today-topic-copy">
        <strong>{item.node.title}</strong>
        <small>{meta}</small>
      </span>
      <Icon name="arrow-up-right" size={15} />
    </Link>
  )
}

function TodayEmpty({ text }: { text: string }) {
  return <p className="today-inline-empty"><Icon name="compass" size={15} /> {text}</p>
}

function TodaySkeleton() {
  return (
    <div className="today-loading" aria-label="Загрузка сегодняшнего плана">
      <span /><span /><span /><span /><span /><span />
    </div>
  )
}
