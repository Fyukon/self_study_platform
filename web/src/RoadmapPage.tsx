import { FormEvent, useEffect, useMemo, useState } from 'react'
import { api } from './api'
import { calculateProgress, RoadmapTree, statusLabels } from './RoadmapTree'
import type { Direction, NodeType, RoadmapNode, RoadmapStatus } from './types'
import { Icon, type IconName } from './icons'
import { useParams, useSearchParams } from 'react-router-dom'

const nodeTypeLabels: Record<NodeType, string> = {
  section: 'Раздел',
  concept: 'Тема',
  practice: 'Практика',
  project: 'Проект',
  checkpoint: 'Проверка',
}

const nodeTypeIcons: Record<NodeType, IconName> = {
  section: 'roadmap',
  concept: 'book',
  practice: 'chart',
  project: 'briefcase',
  checkpoint: 'compass',
}

const statusReadiness: Record<RoadmapStatus, number> = {
  not_started: 0,
  learning: 35,
  practicing: 70,
  understood: 100,
}

const directionIcons: Record<string, IconName> = {
  server: 'server',
  backend: 'server',
  go: 'server',
  linux: 'compass',
  devops: 'chart',
  architecture: 'roadmap',
  database: 'chart',
}

function directionIcon(value: string): IconName {
  return directionIcons[value.trim().toLocaleLowerCase('en-US')] ?? 'compass'
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Не удалось выполнить запрос'
}

function findNode(nodes: RoadmapNode[], id: number | null): RoadmapNode | null {
  if (id === null) return null
  for (const node of nodes) {
    if (node.id === id) return node
    const found = findNode(node.children ?? [], id)
    if (found) return found
  }
  return null
}

function replaceNode(nodes: RoadmapNode[], updated: RoadmapNode): RoadmapNode[] {
  return nodes.map((node) => {
    if (node.id === updated.id) return { ...node, ...updated, children: updated.children ?? node.children }
    return node.children ? { ...node, children: replaceNode(node.children, updated) } : node
  })
}

function removeNode(nodes: RoadmapNode[], id: number): RoadmapNode[] {
  return nodes.filter((node) => node.id !== id).map((node) => (
    node.children ? { ...node, children: removeNode(node.children, id) } : node
  ))
}

function flattenNodes(nodes: RoadmapNode[]): RoadmapNode[] {
  return nodes.flatMap((node) => [node, ...flattenNodes(node.children ?? [])])
}

function directionSlug(title: string): string {
  return title.trim().toLocaleLowerCase('ru-RU').replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-|-$/g, '')
}

function decodeReference(reference: string): string {
  try {
    return decodeURIComponent(reference)
  } catch {
    return reference
  }
}

export function directionIDFromReference(items: Direction[], reference: string | null): number | null {
  if (!reference) return null
  const decodedReference = decodeReference(reference)
  const numericID = Number(decodedReference)
  if (Number.isSafeInteger(numericID) && numericID > 0 && items.some((item) => item.id === numericID)) {
    return numericID
  }
  return items.find((item) => directionSlug(item.title) === directionSlug(decodedReference))?.id ?? null
}

export function RoadmapPage() {
  const { directionId: pathDirection } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const requestedDirection = searchParams.get('direction') ?? pathDirection ?? null
  const [directions, setDirections] = useState<Direction[]>([])
  const [directionId, setDirectionId] = useState<number | null>(null)
  const [nodes, setNodes] = useState<RoadmapNode[]>([])
  const [selectedNodeId, setSelectedNodeId] = useState<number | null>(null)
  const [directionsLoading, setDirectionsLoading] = useState(true)
  const [roadmapLoading, setRoadmapLoading] = useState(false)
  const [directionsError, setDirectionsError] = useState('')
  const [roadmapError, setRoadmapError] = useState('')
  const [directionsReload, setDirectionsReload] = useState(0)
  const [roadmapReload, setRoadmapReload] = useState(0)
  const [showDirectionForm, setShowDirectionForm] = useState(false)
  const [newNodeParent, setNewNodeParent] = useState<number | null | undefined>(undefined)
  const [topicFullscreen, setTopicFullscreen] = useState(false)

  useEffect(() => {
    const controller = new AbortController()
    setDirectionsLoading(true)
    setDirectionsError('')
    api.directions(controller.signal)
      .then((items) => {
        setDirections(items)
        const requestedID = directionIDFromReference(items, requestedDirection)
        setDirectionId((current) => {
          if (requestedID !== null) return requestedID
          if (current !== null && items.some((item) => item.id === current)) return current
          return items.find((item) => !item.is_archived)?.id ?? items[0]?.id ?? null
        })
      })
      .catch((error) => {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          setDirectionsError(errorMessage(error))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setDirectionsLoading(false)
      })
    return () => controller.abort()
  }, [directionsReload, requestedDirection])

  useEffect(() => {
    if (directionId === null) {
      setNodes([])
      return
    }
    const controller = new AbortController()
    setRoadmapLoading(true)
    setRoadmapError('')
    api.roadmap(directionId, controller.signal)
      .then(setNodes)
      .catch((error) => {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          setRoadmapError(errorMessage(error))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setRoadmapLoading(false)
      })
    return () => controller.abort()
  }, [directionId, roadmapReload])

  const activeDirections = directions.filter((direction) => !direction.is_archived)
  const archivedDirections = directions.filter((direction) => direction.is_archived)
  const direction = directions.find((item) => item.id === directionId) ?? null
  const selectedNode = useMemo(() => findNode(nodes, selectedNodeId), [nodes, selectedNodeId])
  const progress = useMemo(() => calculateProgress(nodes), [nodes])
  const roadmapStats = useMemo(() => {
    const allNodes = flattenNodes(nodes)
    const topics = allNodes.filter((node) => node.node_type !== 'section')
    return {
      topics: topics.length,
      sections: allNodes.length - topics.length,
      active: topics.filter((node) => node.status === 'learning' || node.status === 'practicing').length,
      review: topics.filter((node) => node.needs_review).length,
    }
  }, [nodes])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (topicFullscreen) {
          event.preventDefault()
          setTopicFullscreen(false)
        } else if (newNodeParent !== undefined) {
          event.preventDefault()
          setNewNodeParent(undefined)
        } else if (showDirectionForm) {
          event.preventDefault()
          setShowDirectionForm(false)
        } else if (selectedNodeId !== null) {
          event.preventDefault()
          setSelectedNodeId(null)
        }
        return
      }

      const target = event.target
      if (target instanceof HTMLElement && target.matches('input, textarea, select, button, [contenteditable="true"]')) return
      if (event.metaKey || event.ctrlKey || event.altKey || event.repeat) return

      if (event.key.toLowerCase() === 'n' && direction) {
        event.preventDefault()
        setNewNodeParent(null)
      } else if (event.key.toLowerCase() === 'r' && directionId !== null) {
        event.preventDefault()
        setRoadmapReload((value) => value + 1)
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [direction, directionId, newNodeParent, selectedNodeId, showDirectionForm, topicFullscreen])

  useEffect(() => {
    if (!topicFullscreen) return
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = previousOverflow }
  }, [topicFullscreen])

  const selectDirection = (id: number) => {
    // A click on the active direction is a no-op: changing search params here
    // would refetch directions and make an already loaded tree flicker.
    if (id === directionId) return
    setDirectionId(id)
    setNodes([])
    setSelectedNodeId(null)
    setTopicFullscreen(false)
    setSearchParams({ direction: String(id) })
  }

  return (
    <section className="roadmap-page" aria-labelledby="roadmap-title">
      <aside className="directions-panel">
        <div className="panel-heading">
          <div>
            <p className="eyebrow">Пространство</p>
            <h2>Направления</h2>
          </div>
          <button
            className="icon-button"
            type="button"
            aria-label="Создать направление"
            onClick={() => setShowDirectionForm(!showDirectionForm)}
          >
            <Icon name="plus" size={17} />
          </button>
        </div>

        {showDirectionForm && (
          <DirectionForm
            onCancel={() => setShowDirectionForm(false)}
            onCreated={(created) => {
              setDirections((items) => [...items, created])
              selectDirection(created.id)
              setShowDirectionForm(false)
            }}
          />
        )}

        {directionsLoading ? (
          <div className="loading-list" aria-label="Загрузка направлений">
            <span /><span /><span />
          </div>
        ) : directionsError ? (
          <ErrorState message={directionsError} onRetry={() => setDirectionsReload((value) => value + 1)} />
        ) : directions.length === 0 ? (
          <div className="compact-empty">
            <p>Направлений пока нет.</p>
            <button className="text-button" type="button" onClick={() => setShowDirectionForm(true)}>
              Создать первое
            </button>
          </div>
        ) : (
          <>
            <div className="direction-list">
              {activeDirections.map((item) => (
                <DirectionButton
                  key={item.id}
                  direction={item}
                  active={item.id === directionId}
                  onClick={() => selectDirection(item.id)}
                />
              ))}
            </div>
            {archivedDirections.length > 0 && (
              <details className="archived-directions">
                <summary>Архив · {archivedDirections.length}</summary>
                <div className="direction-list">
                  {archivedDirections.map((item) => (
                    <DirectionButton
                      key={item.id}
                      direction={item}
                      active={item.id === directionId}
                      onClick={() => selectDirection(item.id)}
                    />
                  ))}
                </div>
              </details>
            )}
          </>
        )}
      </aside>

      <div className="roadmap-center">
        <header className="roadmap-header">
          <div>
            <p className="eyebrow">Roadmap</p>
            <h1 id="roadmap-title">{direction?.title ?? 'Карта обучения'}</h1>
            {direction?.description && <p className="roadmap-description">{direction.description}</p>}
            {direction && <p className="keyboard-hints">Esc закрыть · N новая тема · R обновить</p>}
          </div>
          {direction && (
            <button className="button secondary" type="button" onClick={() => setNewNodeParent(null)}>
              <Icon name="plus" size={16} /> Новая тема
            </button>
          )}
        </header>

        {direction && !roadmapLoading && !roadmapError && nodes.length > 0 && (
          <div className="progress-summary">
            <div className="progress-overview">
              <div
                className="overview-ring"
                style={{ background: `conic-gradient(var(--accent) ${progress.percent * 3.6}deg, var(--panel-strong) 0)` }}
                role="img"
                aria-label={`Освоено ${progress.percent}%`}
              >
                <span><strong>{progress.percent}%</strong><small>освоено</small></span>
              </div>
              <div className="progress-overview-copy">
                <div className="progress-summary-line">
                  <span>Освоено {progress.understood} из {progress.total}</span>
                  <strong>{progress.percent}%</strong>
                </div>
                <div
                  className="progress-track"
                  role="progressbar"
                  aria-label="Освоенные темы"
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={progress.percent}
                >
                  <span style={{ width: `${progress.percent}%` }} />
                </div>
                <div className="roadmap-stats" aria-label="Статистика roadmap">
                  <span><strong>{roadmapStats.topics}</strong> тем</span>
                  <span><strong>{roadmapStats.sections}</strong> разделов</span>
                  <span><strong>{roadmapStats.active}</strong> в работе</span>
                  <span><strong>{roadmapStats.review}</strong> повторить</span>
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="roadmap-scroll">
          {!direction ? (
            <EmptyState
              title="Выберите направление"
              text="Создайте направление, чтобы добавить первую карту обучения."
            />
          ) : roadmapLoading ? (
            <TreeSkeleton />
          ) : roadmapError ? (
            <ErrorState message={roadmapError} onRetry={() => setRoadmapReload((value) => value + 1)} />
          ) : nodes.length === 0 ? (
            <EmptyState
              title="Roadmap пока пуст"
              text="Добавьте первый раздел или тему — вложенность можно наращивать позже."
              action={<button className="button primary" type="button" onClick={() => setNewNodeParent(null)}>Добавить тему</button>}
            />
          ) : (
            <RoadmapTree
              nodes={nodes}
              selectedId={selectedNodeId}
              onSelect={(node) => {
                setSelectedNodeId(node.id)
                setTopicFullscreen(false)
              }}
            />
          )}
        </div>
      </div>

      <aside className="details-panel">
        {selectedNode ? (
          <NodeDetails
            key={selectedNode.id}
            node={selectedNode}
            allNodes={flattenNodes(nodes)}
            expanded={topicFullscreen}
            onToggleFullscreen={() => setTopicFullscreen((value) => !value)}
            onAddChild={() => setNewNodeParent(selectedNode.id)}
            onSaved={(updated) => setNodes((items) => replaceNode(items, updated))}
            onDependenciesChanged={() => setRoadmapReload((value) => value + 1)}
            onDeleted={(id) => {
              setNodes((items) => removeNode(items, id))
              setSelectedNodeId(null)
              setTopicFullscreen(false)
            }}
          />
        ) : (
          <div className="details-empty">
            <Icon name="compass" size={30} />
            <h2>Выберите тему</h2>
            <p>Здесь можно изменить прогресс и записать следующий конкретный шаг.</p>
          </div>
        )}
      </aside>

      {newNodeParent !== undefined && direction && (
        <NodeForm
          directionId={direction.id}
          parentId={newNodeParent}
          parentTitle={newNodeParent === null ? '' : findNode(nodes, newNodeParent)?.title ?? ''}
          onCancel={() => setNewNodeParent(undefined)}
          onCreated={(created) => {
            setNewNodeParent(undefined)
            setRoadmapReload((value) => value + 1)
            setSelectedNodeId(created.id)
          }}
        />
      )}
    </section>
  )
}

function DirectionButton({
  direction,
  active,
  onClick,
}: {
  direction: Direction
  active: boolean
  onClick: () => void
}) {
  return (
    <button className={`direction-button ${active ? 'active' : ''}`} type="button" onClick={onClick}>
      <span className="direction-icon"><Icon name={directionIcon(direction.icon)} size={17} /></span>
      <span>
        <strong>{direction.title}</strong>
        {direction.description && <small>{direction.description}</small>}
      </span>
    </button>
  )
}

function DirectionForm({ onCancel, onCreated }: { onCancel: () => void; onCreated: (item: Direction) => void }) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      onCreated(await api.createDirection({ title: title.trim(), description: description.trim() }))
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form className="direction-form" onSubmit={submit}>
      <label>
        Название
        <input autoFocus required maxLength={120} value={title} onChange={(event) => setTitle(event.target.value)} />
      </label>
      <label>
        Описание
        <textarea rows={2} maxLength={500} value={description} onChange={(event) => setDescription(event.target.value)} />
      </label>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="form-actions">
        <button className="text-button" type="button" onClick={onCancel}>Отмена</button>
        <button className="button primary small" type="submit" disabled={saving}>{saving ? 'Создаём…' : 'Создать'}</button>
      </div>
    </form>
  )
}

function NodeForm({
  directionId,
  parentId,
  parentTitle,
  onCancel,
  onCreated,
}: {
  directionId: number
  parentId: number | null
  parentTitle: string
  onCancel: () => void
  onCreated: (node: RoadmapNode) => void
}) {
  const [title, setTitle] = useState('')
  const [nodeType, setNodeType] = useState<NodeType>('concept')
  const [description, setDescription] = useState('')
  const [nextAction, setNextAction] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      onCreated(await api.createNode({
        direction_id: directionId,
        parent_id: parentId,
        title: title.trim(),
        node_type: nodeType,
        description: description.trim(),
        next_action: nextAction.trim(),
      }))
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-backdrop">
      <form className="modal-card" role="dialog" aria-modal="true" aria-labelledby="node-form-title" onSubmit={submit}>
        <div className="modal-heading">
          <div>
            <p className="eyebrow">{parentId === null ? 'В корне roadmap' : `Внутри «${parentTitle}»`}</p>
            <h2 id="node-form-title">Новая тема</h2>
          </div>
          <button className="icon-button" type="button" aria-label="Закрыть" onClick={onCancel}><Icon name="close" size={16} /></button>
        </div>
        <label>
          Название
          <input autoFocus required maxLength={180} value={title} onChange={(event) => setTitle(event.target.value)} />
        </label>
        <label>
          Тип
          <select value={nodeType} onChange={(event) => setNodeType(event.target.value as NodeType)}>
            {Object.entries(nodeTypeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <label>
          Что изучить
          <textarea
            rows={4}
            maxLength={4000}
            placeholder="Ключевые понятия, практика и ожидаемый результат"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </label>
        <label>
          Следующий шаг
          <input
            maxLength={1000}
            placeholder="Например: решить 3 задачи на тему"
            value={nextAction}
            onChange={(event) => setNextAction(event.target.value)}
          />
        </label>
        {error && <p className="form-error" role="alert">{error}</p>}
        <div className="form-actions">
          <button className="button secondary" type="button" onClick={onCancel}>Отмена</button>
          <button className="button primary" type="submit" disabled={saving}>{saving ? 'Добавляем…' : 'Добавить'}</button>
        </div>
      </form>
    </div>
  )
}

function NodeDetails({
  node,
  allNodes,
  expanded,
  onToggleFullscreen,
  onAddChild,
  onSaved,
  onDependenciesChanged,
  onDeleted,
}: {
  node: RoadmapNode
  allNodes: RoadmapNode[]
  expanded: boolean
  onToggleFullscreen: () => void
  onAddChild: () => void
  onSaved: (node: RoadmapNode) => void
  onDependenciesChanged: () => void
  onDeleted: (id: number) => void
}) {
  const [status, setStatus] = useState<RoadmapStatus>(node.status)
  const [confidence, setConfidence] = useState(node.confidence || 1)
  const [needsReview, setNeedsReview] = useState(node.needs_review)
  const [nextAction, setNextAction] = useState(node.next_action)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [dependencyNodeId, setDependencyNodeId] = useState<number | null>(null)
  const [changingDependency, setChangingDependency] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setSaved(false)
    setError('')
    try {
      const updated = await api.updateNode(node.id, {
        status,
        confidence,
        needs_review: needsReview,
        next_action: nextAction.trim(),
      })
      onSaved(updated)
      setSaved(true)
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!window.confirm(`Удалить «${node.title}» и вложенные темы?`)) return
    setDeleting(true)
    setError('')
    try {
      await api.deleteNode(node.id)
      onDeleted(node.id)
    } catch (requestError) {
      setError(errorMessage(requestError))
      setDeleting(false)
    }
  }

  const addDependency = async () => {
    if (dependencyNodeId === null) return
    setChangingDependency(true)
    setError('')
    try {
      await api.createDependency(node.id, dependencyNodeId)
      setDependencyNodeId(null)
      onDependenciesChanged()
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setChangingDependency(false)
    }
  }

  const removeDependency = async (dependencyId: number) => {
    setChangingDependency(true)
    setError('')
    try {
      await api.deleteDependency(node.id, dependencyId)
      onDependenciesChanged()
    } catch (requestError) {
      setError(errorMessage(requestError))
    } finally {
      setChangingDependency(false)
    }
  }

  const dependencyOptions = allNodes.filter((candidate) =>
    candidate.id !== node.id && !node.dependencies.some((item) => item.depends_on_node_id === candidate.id),
  )

  return (
    <div
      className={expanded ? 'topic-fullscreen-shell' : undefined}
      role={expanded ? 'dialog' : undefined}
      aria-modal={expanded ? true : undefined}
      aria-labelledby={expanded ? `topic-title-${node.id}` : undefined}
    >
      {expanded && (
        <button
          className="topic-fullscreen-backdrop"
          type="button"
          aria-label="Закрыть полноэкранную карточку"
          onClick={onToggleFullscreen}
        />
      )}
      <form className={`details-form topic-card${expanded ? ' topic-card-expanded' : ''}`} onSubmit={submit}>
        <div className="topic-hero">
          <div className={`topic-icon topic-icon-${node.node_type}`}>
            <Icon name={nodeTypeIcons[node.node_type]} size={23} />
          </div>
          <div className="topic-hero-copy">
            <span className="topic-kicker">Карточка обучения</span>
            <div className="topic-state-row">
              <span className={`status-badge status-${node.status}`}>{statusLabels[node.status]}</span>
              {node.needs_review && <span className="review-badge">Повторить</span>}
            </div>
          </div>
          <div
            className="topic-ring"
            style={{ background: `conic-gradient(var(--accent) ${statusReadiness[node.status] * 3.6}deg, var(--panel-strong) 0)` }}
            role="img"
            aria-label={`Готовность темы ${statusReadiness[node.status]}%`}
          >
            <span><strong>{statusReadiness[node.status]}%</strong><small>готово</small></span>
          </div>
          <button
            className="icon-button topic-expand-button"
            type="button"
            aria-label={expanded ? 'Свернуть карточку' : 'Развернуть карточку на весь экран'}
            title={expanded ? 'Свернуть карточку' : 'На весь экран'}
            onClick={onToggleFullscreen}
          >
            <Icon name={expanded ? 'compress' : 'expand'} size={16} />
          </button>
        </div>

      <div className="details-heading">
        <p className="eyebrow">{nodeTypeLabels[node.node_type]}</p>
        <h2 id={`topic-title-${node.id}`}>{node.title}</h2>
      </div>

      <section className="topic-study" aria-labelledby={`topic-study-${node.id}`}>
        <div className="topic-section-heading">
          <Icon name="book" size={15} />
          <h3 id={`topic-study-${node.id}`}>Что изучить</h3>
        </div>
        <p className="topic-description">
          {node.description || 'Добавьте описание: ключевые понятия, практику и ожидаемый результат.'}
        </p>
      </section>

      <div className="topic-metrics" aria-label="Метрики темы">
        <div className="topic-metric">
          <span>Уверенность</span>
          <strong>{node.confidence}/5</strong>
          <span className="metric-track"><span style={{ width: `${node.confidence * 20}%` }} /></span>
        </div>
        <div className="topic-metric">
          <span>{node.node_type === 'section' ? 'Подтемы' : 'Оценка'}</span>
          <strong>{node.node_type === 'section' ? node.children?.length ?? 0 : node.estimated_hours ? `${node.estimated_hours} ч` : '—'}</strong>
          <small>{node.node_type === 'section' ? 'внутри раздела' : 'учебного времени'}</small>
        </div>
        <div className="topic-metric">
          <span>Фокус</span>
          <strong>{node.next_action ? 'Есть' : 'Нужен'}</strong>
          <small>{node.next_action ? 'следующий шаг' : 'добавьте действие'}</small>
        </div>
      </div>

      {node.next_action && (
        <div className="topic-next-action">
          <span>Следующий практический шаг</span>
          <strong>{node.next_action}</strong>
        </div>
      )}

      <label>
        Статус
        <select value={status} onChange={(event) => setStatus(event.target.value as RoadmapStatus)}>
          {Object.entries(statusLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>

      <label>
        <span className="range-label">Уверенность <strong>{confidence}/5</strong></span>
        <input
          className="confidence-range"
          type="range"
          min={1}
          max={5}
          step={1}
          value={confidence}
          onChange={(event) => setConfidence(Number(event.target.value))}
        />
      </label>

      <label className="checkbox-row">
        <input type="checkbox" checked={needsReview} onChange={(event) => setNeedsReview(event.target.checked)} />
        <span>
          <strong>Нужно повторить</strong>
          <small>Независимо от основного статуса</small>
        </span>
      </label>

      <label>
        Следующее действие
        <textarea
          rows={4}
          maxLength={1000}
          placeholder="Например: написать тест для обработчика"
          value={nextAction}
          onChange={(event) => setNextAction(event.target.value)}
        />
      </label>

      <section className="dependencies-section" aria-labelledby="dependencies-title">
        <h3 id="dependencies-title">Зависимости</h3>
        {node.dependencies.length === 0 ? (
          <p className="muted-text">Нет зависимостей</p>
        ) : (
          <ul className="dependency-list">
            {node.dependencies.map((dependency) => (
              <li key={dependency.id}>
                <span>{allNodes.find((item) => item.id === dependency.depends_on_node_id)?.title ?? `Тема #${dependency.depends_on_node_id}`}</span>
                <button
                  type="button"
                  className="icon-button"
                  disabled={changingDependency}
                  aria-label="Удалить зависимость"
                  onClick={() => removeDependency(dependency.id)}
                >
                  <Icon name="close" size={14} />
                </button>
              </li>
            ))}
          </ul>
        )}
        {dependencyOptions.length > 0 && (
          <div className="dependency-add">
            <label>
              Зависит от темы
              <select
                value={dependencyNodeId ?? ''}
                onChange={(event) => setDependencyNodeId(event.target.value ? Number(event.target.value) : null)}
              >
                <option value="">Выберите тему</option>
                {dependencyOptions.map((candidate) => (
                  <option key={candidate.id} value={candidate.id}>{candidate.title}</option>
                ))}
              </select>
            </label>
            <button
              className="button secondary small"
              type="button"
              disabled={dependencyNodeId === null || changingDependency}
              onClick={addDependency}
            >
              Добавить
            </button>
          </div>
        )}
      </section>

      {error && <p className="form-error" role="alert">{error}</p>}
      <p className="save-status" aria-live="polite">{saved ? 'Изменения сохранены' : ''}</p>

      <button className="button primary full" type="submit" disabled={saving}>{saving ? 'Сохраняем…' : 'Сохранить изменения'}</button>
      <button className="button secondary full" type="button" onClick={onAddChild}><Icon name="plus" size={16} /> Добавить вложенную тему</button>
        <button className="danger-button" type="button" onClick={remove} disabled={deleting}>{deleting ? 'Удаляем…' : 'Удалить тему'}</button>
      </form>
    </div>
  )
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="error-state" role="alert">
      <strong>Не удалось загрузить данные</strong>
      <p>{message}</p>
      <button className="button secondary small" type="button" onClick={onRetry}>Повторить</button>
    </div>
  )
}

function EmptyState({ title, text, action }: { title: string; text: string; action?: React.ReactNode }) {
  return (
    <div className="empty-card">
      <div className="empty-symbol"><Icon name="compass" size={24} /></div>
      <h2>{title}</h2>
      <p>{text}</p>
      {action}
    </div>
  )
}

function TreeSkeleton() {
  return (
    <div className="tree-skeleton" aria-label="Загрузка roadmap">
      <span /><span /><span /><span /><span />
    </div>
  )
}
