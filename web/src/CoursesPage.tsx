import { FormEvent, useEffect, useState } from 'react'
import { api } from './api'
import { flattenNodes } from './RoadmapTree'
import { Icon } from './icons'
import type {
  Course,
  CourseImportPreview,
  CourseModule,
  CourseModuleStatus,
  CourseResource,
  CourseSummary,
  CreateResource,
  Direction,
  ResourceType,
  RoadmapNode,
} from './types'

const resourceTypeLabels: Record<ResourceType, string> = {
  book: 'Книга',
  video: 'Видео',
  repository: 'Репозиторий',
  article: 'Статья',
  link: 'Ссылка',
  local_file: 'Локальный файл',
}

const moduleStatusLabels: Record<CourseModuleStatus, string> = {
  not_started: 'Не начат',
  in_progress: 'В процессе',
  completed: 'Завершён',
}

const resourceTypes = Object.keys(resourceTypeLabels) as ResourceType[]

const markdownExample = `# Go для backend-разработки

Практический курс с книгами и исходниками.

## Основы языка

Типы, функции и работа с ошибками.

- [book] The Go Programming Language — https://example.com/go-book
- [repo] Практические примеры — https://github.com/example/go

## Локальные заметки

- [local] Мой конспект — /home/user/notes/go.md`

const jsonExample = `{
  "title": "Go для backend-разработки",
  "description": "Практический курс",
  "provider": "Self study",
  "source_url": "https://example.com/course",
  "modules": [
    {
      "title": "Основы языка",
      "description": "Типы и функции",
      "resources": [
        { "title": "Книга", "resource_type": "book", "url": "https://example.com/book" }
      ]
    }
  ]
}`

export function courseResourceHref(resource: Pick<CourseResource, 'resource_type' | 'url' | 'local_path'>): string {
  return resource.resource_type === 'local_file' ? `file://${resource.local_path}` : resource.url
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Не удалось выполнить запрос'
}

interface RoadmapOption {
  direction: Direction
  node: RoadmapNode
}

export function CoursesPage() {
  const [courses, setCourses] = useState<CourseSummary[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selectedCourse, setSelectedCourse] = useState<Course | null>(null)
  const [roadmapOptions, setRoadmapOptions] = useState<RoadmapOption[]>([])
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [error, setError] = useState('')
  const [actionError, setActionError] = useState('')
  const [showCourseForm, setShowCourseForm] = useState(false)
  const [showImport, setShowImport] = useState(false)

  const refresh = async (id = selectedId) => {
    const [courseItems, course] = await Promise.all([
      api.courses(),
      id === null ? Promise.resolve(null) : api.course(id),
    ])
    setCourses(courseItems)
    if (course) setSelectedCourse(course)
    return course
  }

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setError('')
    api.courses(controller.signal)
      .then((items) => {
        setCourses(items)
        setSelectedId((current) => current ?? items.find((course) => !course.is_archived)?.id ?? items[0]?.id ?? null)
      })
      .catch((requestError) => {
        if (!(requestError instanceof DOMException && requestError.name === 'AbortError')) setError(errorMessage(requestError))
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [])

  useEffect(() => {
    if (selectedId === null) {
      setSelectedCourse(null)
      return
    }
    const controller = new AbortController()
    setDetailLoading(true)
    setActionError('')
    api.course(selectedId, controller.signal)
      .then(setSelectedCourse)
      .catch((requestError) => {
        if (!(requestError instanceof DOMException && requestError.name === 'AbortError')) setActionError(errorMessage(requestError))
      })
      .finally(() => {
        if (!controller.signal.aborted) setDetailLoading(false)
      })
    return () => controller.abort()
  }, [selectedId])

  useEffect(() => {
    const controller = new AbortController()
    api.directions(controller.signal)
      .then((directions) => Promise.all(
        directions
          .filter((direction) => !direction.is_archived)
          .map(async (direction) => ({ direction, nodes: await api.roadmap(direction.id, controller.signal) })),
      ))
      .then((snapshots) => setRoadmapOptions(snapshots.flatMap(({ direction, nodes }) =>
        flattenNodes(nodes)
          .filter((node) => node.node_type !== 'section')
          .map((node) => ({ direction, node })),
      )))
      .catch((requestError) => {
        if (!(requestError instanceof DOMException && requestError.name === 'AbortError')) setActionError(errorMessage(requestError))
      })
    return () => controller.abort()
  }, [])

  const createCourse = async (data: { title: string; description: string; provider: string; source_url: string }) => {
    setActionError('')
    try {
      const created = await api.createCourse(data)
      setCourses((items) => [...items, {
        ...created,
        module_count: 0,
        resource_count: 0,
      }])
      setSelectedId(created.id)
      setShowCourseForm(false)
    } catch (requestError) {
      setActionError(errorMessage(requestError))
    }
  }

  const importCourse = async (preview: CourseImportPreview) => {
    setActionError('')
    try {
      const imported = await api.importCourse(preview.preview_id)
      const items = await api.courses()
      setCourses(items)
      setSelectedId(imported.id)
      setShowImport(false)
    } catch (requestError) {
      setActionError(errorMessage(requestError))
    }
  }

  return (
    <section className="courses-page" aria-labelledby="courses-title">
      <aside className="courses-sidebar">
        <div className="panel-heading">
          <div>
            <p className="eyebrow">Библиотека</p>
            <h2>Курсы</h2>
          </div>
          <button className="icon-button" type="button" aria-label="Создать курс" onClick={() => setShowCourseForm((value) => !value)}>
            <Icon name="plus" size={17} />
          </button>
        </div>

        {showCourseForm && <CourseCreateForm onCancel={() => setShowCourseForm(false)} onSubmit={createCourse} />}

        {loading ? <div className="loading-list"><span /><span /><span /></div> : courses.length === 0 ? (
          <div className="compact-empty"><p>Курсов пока нет.</p><button className="text-button" type="button" onClick={() => setShowCourseForm(true)}>Создать первый</button></div>
        ) : (
          <div className="course-list" role="listbox" aria-label="Курсы">
            {courses.map((course) => (
              <button
                className={`course-list-item ${selectedId === course.id ? 'active' : ''} ${course.is_archived ? 'archived' : ''}`}
                key={course.id}
                type="button"
                role="option"
                aria-selected={selectedId === course.id}
                onClick={() => setSelectedId(course.id)}
              >
                <span className="course-list-icon"><Icon name="book" size={16} /></span>
                <span className="course-list-copy">
                  <strong>{course.title}</strong>
                  <small>{course.module_count} модулей · {course.resource_count} материалов</small>
                </span>
              </button>
            ))}
          </div>
        )}

        <button className="course-import-trigger" type="button" onClick={() => setShowImport((value) => !value)}>
          <Icon name="arrow-up-right" size={15} /> Импорт из Markdown / JSON
        </button>
      </aside>

      <main className="courses-main">
        <header className="courses-header">
          <div>
            <p className="eyebrow">Курсы и материалы</p>
            <h1 id="courses-title">Учитесь по своему маршруту</h1>
            <p>Собирайте модули, прикрепляйте книги и репозитории, связывайте обучение с темами roadmap.</p>
          </div>
          {selectedCourse && <span className="course-header-count">{selectedCourse.modules.length} модулей</span>}
        </header>

        {showImport && <CourseImportPanel onClose={() => setShowImport(false)} onImported={importCourse} />}
        {actionError && <div className="courses-alert" role="alert"><Icon name="compass" size={16} /> {actionError}</div>}
        {error ? (
          <div className="error-state courses-state"><strong>Не удалось загрузить курсы</strong><p>{error}</p><button className="button secondary" type="button" onClick={() => window.location.reload()}>Повторить</button></div>
        ) : detailLoading ? (
          <div className="courses-detail-skeleton"><span /><span /><span /></div>
        ) : selectedCourse ? (
          <CourseDetail course={selectedCourse} roadmapOptions={roadmapOptions} onRefresh={() => refresh()} onError={setActionError} onCourseUpdated={(course) => {
            setSelectedCourse(course)
            setCourses((items) => items.map((item) => item.id === course.id ? { ...item, ...course, module_count: course.modules.length, resource_count: course.modules.reduce((count, module) => count + module.resources.length, 0) } : item))
          }} />
        ) : (
          <div className="empty-card courses-empty"><div className="empty-symbol"><Icon name="book" size={23} /></div><h2>Добавьте первый курс</h2><p>Начните с ручной записи или вставьте Markdown/JSON и проверьте его перед импортом.</p><button className="button primary" type="button" onClick={() => setShowCourseForm(true)}>Создать курс</button></div>
        )}
      </main>
    </section>
  )
}

function CourseCreateForm({ onCancel, onSubmit }: { onCancel: () => void; onSubmit: (data: { title: string; description: string; provider: string; source_url: string }) => void }) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [provider, setProvider] = useState('')
  const [sourceUrl, setSourceUrl] = useState('')
  const submit = (event: FormEvent) => {
    event.preventDefault()
    onSubmit({ title, description, provider, source_url: sourceUrl })
  }
  return (
    <form className="course-create-form" onSubmit={submit}>
      <label>Название<input required value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Например, Go backend" /></label>
      <label>Провайдер<input value={provider} onChange={(event) => setProvider(event.target.value)} placeholder="Self study" /></label>
      <label>Ссылка на курс<input type="url" value={sourceUrl} onChange={(event) => setSourceUrl(event.target.value)} placeholder="https://..." /></label>
      <label>Описание<textarea rows={3} value={description} onChange={(event) => setDescription(event.target.value)} /></label>
      <div className="form-actions"><button className="button secondary small" type="button" onClick={onCancel}>Отмена</button><button className="button primary small" type="submit">Создать</button></div>
    </form>
  )
}

function CourseDetail({
  course,
  roadmapOptions,
  onRefresh,
  onError,
  onCourseUpdated,
}: {
  course: Course
  roadmapOptions: RoadmapOption[]
  onRefresh: () => Promise<unknown>
  onError: (message: string) => void
  onCourseUpdated: (course: Course) => void
}) {
  const [editing, setEditing] = useState(false)
  const [showModuleForm, setShowModuleForm] = useState(false)
  const [title, setTitle] = useState(course.title)
  const [description, setDescription] = useState(course.description)
  const [provider, setProvider] = useState(course.provider)
  const [sourceUrl, setSourceUrl] = useState(course.source_url)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setTitle(course.title)
    setDescription(course.description)
    setProvider(course.provider)
    setSourceUrl(course.source_url)
    setEditing(false)
  }, [course.id, course.title, course.description, course.provider, course.source_url])

  const saveCourse = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    onError('')
    try {
      const updated = await api.updateCourse(course.id, { title, description, provider, source_url: sourceUrl })
      onCourseUpdated(updated)
      setEditing(false)
    } catch (requestError) {
      onError(errorMessage(requestError))
    } finally {
      setSaving(false)
    }
  }

  const archive = async () => {
    try {
      await api.archiveCourse(course.id)
      const refreshed = await onRefresh()
      if (!refreshed) return
    } catch (requestError) {
      onError(errorMessage(requestError))
    }
  }

  return (
    <div className="course-detail">
      <section className="course-overview-card">
        <div className="course-overview-icon"><Icon name="book" size={22} /></div>
        {editing ? (
          <form className="course-edit-form" onSubmit={saveCourse}>
            <label>Название<input required value={title} onChange={(event) => setTitle(event.target.value)} /></label>
            <div className="course-edit-grid"><label>Провайдер<input value={provider} onChange={(event) => setProvider(event.target.value)} /></label><label>Ссылка<input type="url" value={sourceUrl} onChange={(event) => setSourceUrl(event.target.value)} /></label></div>
            <label>Описание<textarea rows={3} value={description} onChange={(event) => setDescription(event.target.value)} /></label>
            <div className="form-actions"><button className="button secondary small" type="button" onClick={() => setEditing(false)}>Отмена</button><button className="button primary small" type="submit" disabled={saving}>{saving ? 'Сохраняем…' : 'Сохранить'}</button></div>
          </form>
        ) : (
          <div className="course-overview-copy">
            <div className="course-overview-heading"><div><p className="eyebrow">{course.is_archived ? 'Архивный курс' : course.provider || 'Личный курс'}</p><h2>{course.title}</h2></div><div className="course-overview-actions"><button className="text-button" type="button" onClick={() => setEditing(true)}>Изменить</button>{!course.is_archived && <button className="danger-button" type="button" onClick={archive}>В архив</button>}</div></div>
            {course.description && <p>{course.description}</p>}
            {course.source_url && <a className="course-source-link" href={course.source_url} target="_blank" rel="noreferrer"><Icon name="arrow-up-right" size={14} /> Открыть страницу курса</a>}
            {course.is_archived && <button className="button secondary small" type="button" onClick={async () => {
              try { const restored = await api.updateCourse(course.id, { is_archived: false }); onCourseUpdated(restored) } catch (requestError) { onError(errorMessage(requestError)) }
            }}>Вернуть из архива</button>}
          </div>
        )}
      </section>

      <div className="course-section-heading"><div><p className="eyebrow">Содержание</p><h2>Модули курса</h2></div><button className="button primary small" type="button" disabled={course.is_archived} onClick={() => setShowModuleForm((value) => !value)}><Icon name="plus" size={14} /> Добавить модуль</button></div>
      {showModuleForm && <ModuleCreateForm courseId={course.id} onCancel={() => setShowModuleForm(false)} onDone={async () => { setShowModuleForm(false); await onRefresh() }} onError={onError} />}
      {course.modules.length === 0 ? <div className="course-inline-empty">В курсе пока нет модулей. Добавьте первый, чтобы прикрепить материалы.</div> : <div className="module-list">{course.modules.map((module) => <ModuleCard key={module.id} module={module} roadmapOptions={roadmapOptions} disabled={course.is_archived} onRefresh={onRefresh} onError={onError} />)}</div>}
    </div>
  )
}

function ModuleCreateForm({ courseId, onCancel, onDone, onError }: { courseId: number; onCancel: () => void; onDone: () => Promise<void>; onError: (message: string) => void }) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [saving, setSaving] = useState(false)
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try { await api.createCourseModule(courseId, { title, description }); await onDone() } catch (requestError) { onError(errorMessage(requestError)) } finally { setSaving(false) }
  }
  return <form className="module-create-form" onSubmit={submit}><label>Название модуля<input required value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Например, HTTP и сети" /></label><label>Описание<textarea rows={2} value={description} onChange={(event) => setDescription(event.target.value)} /></label><div className="form-actions"><button className="button secondary small" type="button" onClick={onCancel}>Отмена</button><button className="button primary small" type="submit" disabled={saving}>{saving ? 'Добавляем…' : 'Добавить'}</button></div></form>
}

function ModuleCard({ module, roadmapOptions, disabled, onRefresh, onError }: { module: CourseModule; roadmapOptions: RoadmapOption[]; disabled: boolean; onRefresh: () => Promise<unknown>; onError: (message: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [showResourceForm, setShowResourceForm] = useState(false)
  const [showLinkForm, setShowLinkForm] = useState(false)
  const [title, setTitle] = useState(module.title)
  const [description, setDescription] = useState(module.description)
  const [status, setStatus] = useState<CourseModuleStatus>(module.status)

  useEffect(() => {
    setTitle(module.title)
    setDescription(module.description)
    setStatus(module.status)
  }, [module.id, module.title, module.description, module.status])

  const save = async (event: FormEvent) => {
    event.preventDefault()
    try { await api.updateCourseModule(module.id, { title, description, status }); setEditing(false); await onRefresh() } catch (requestError) { onError(errorMessage(requestError)) }
  }

  const remove = async () => {
    try { await api.deleteCourseModule(module.id); await onRefresh() } catch (requestError) { onError(errorMessage(requestError)) }
  }

  return (
    <section className={`module-card ${module.status === 'completed' ? 'completed' : ''}`}>
      <div className="module-heading">
        <span className="module-number">{String(module.position + 1).padStart(2, '0')}</span>
        {editing ? <form className="module-edit-form" onSubmit={save}><input required value={title} onChange={(event) => setTitle(event.target.value)} /><textarea rows={2} value={description} onChange={(event) => setDescription(event.target.value)} /><select value={status} onChange={(event) => setStatus(event.target.value as CourseModuleStatus)}>{Object.entries(moduleStatusLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><div className="form-actions"><button className="button secondary small" type="button" onClick={() => setEditing(false)}>Отмена</button><button className="button primary small" type="submit">Сохранить</button></div></form> : <div className="module-heading-copy"><div><h3>{module.title}</h3><span className={`module-status module-status-${module.status}`}>{moduleStatusLabels[module.status]}</span></div>{module.description && <p>{module.description}</p>}</div>}
        {!editing && <div className="module-actions"><button className="text-button" type="button" disabled={disabled} onClick={() => setEditing(true)}>Изменить</button><button className="danger-button" type="button" disabled={disabled} onClick={remove}>Удалить</button></div>}
      </div>
      <div className="module-body">
        <div className="module-subheading"><span>Материалы <strong>{module.resources.length}</strong></span><button className="text-button" type="button" disabled={disabled} onClick={() => setShowResourceForm((value) => !value)}><Icon name="plus" size={14} /> Добавить</button></div>
        {showResourceForm && <ResourceForm moduleId={module.id} onCancel={() => setShowResourceForm(false)} onDone={async () => { setShowResourceForm(false); await onRefresh() }} onError={onError} />}
        {module.resources.length > 0 && <div className="resource-list">{module.resources.map((resource) => <ResourceRow key={resource.id} resource={resource} disabled={disabled} onRefresh={onRefresh} onError={onError} />)}</div>}
        {module.resources.length === 0 && !showResourceForm && <p className="module-empty">Добавьте книгу, видео, репозиторий, ссылку или локальный файл.</p>}
        <div className="module-subheading module-links-heading"><span>Связь с roadmap <strong>{module.roadmap_links.length}</strong></span><button className="text-button" type="button" disabled={disabled || roadmapOptions.length === 0} onClick={() => setShowLinkForm((value) => !value)}><Icon name="roadmap" size={14} /> Связать тему</button></div>
        {showLinkForm && <RoadmapLinkForm module={module} options={roadmapOptions} onCancel={() => setShowLinkForm(false)} onDone={async () => { setShowLinkForm(false); await onRefresh() }} onError={onError} />}
        {module.roadmap_links.length > 0 && <div className="roadmap-link-list">{module.roadmap_links.map((link) => <div className="roadmap-link" key={link.id}><Icon name="roadmap" size={14} /><span><strong>{link.node_title}</strong><small>{link.direction_title}</small></span><button className="text-button" type="button" disabled={disabled} aria-label={`Удалить связь с ${link.node_title}`} onClick={async () => { try { await api.deleteRoadmapLink(module.id, link.id); await onRefresh() } catch (requestError) { onError(errorMessage(requestError)) } }}>Убрать</button></div>)}</div>}
        {module.roadmap_links.length === 0 && !showLinkForm && <p className="module-empty">Свяжите модуль с темой, чтобы видеть его в контексте roadmap.</p>}
      </div>
    </section>
  )
}

function ResourceForm({ moduleId, onCancel, onDone, onError }: { moduleId: number; onCancel: () => void; onDone: () => Promise<void>; onError: (message: string) => void }) {
  const [title, setTitle] = useState('')
  const [resourceType, setResourceType] = useState<ResourceType>('link')
  const [location, setLocation] = useState('')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const local = resourceType === 'local_file'
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    const data: CreateResource = { title, resource_type: resourceType, note, ...(local ? { local_path: location } : { url: location }) }
    try { await api.createResource(moduleId, data); await onDone() } catch (requestError) { onError(errorMessage(requestError)) } finally { setSaving(false) }
  }
  return <form className="resource-form" onSubmit={submit}><div className="resource-form-grid"><label>Название<input required value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Например, Go by Example" /></label><label>Тип<select value={resourceType} onChange={(event) => { const value = event.target.value as ResourceType; setResourceType(value); setLocation('') }}>{resourceTypes.map((value) => <option key={value} value={value}>{resourceTypeLabels[value]}</option>)}</select></label></div><label>{local ? 'Путь к файлу' : 'URL'}<input required type={local ? 'text' : 'url'} value={location} onChange={(event) => setLocation(event.target.value)} placeholder={local ? '/home/user/notes.md' : 'https://...'} /></label><label>Заметка<textarea rows={2} value={note} onChange={(event) => setNote(event.target.value)} placeholder="Что прочитать или попробовать" /></label><div className="form-actions"><button className="button secondary small" type="button" onClick={onCancel}>Отмена</button><button className="button primary small" type="submit" disabled={saving}>{saving ? 'Добавляем…' : 'Добавить материал'}</button></div></form>
}

function ResourceRow({ resource, disabled, onRefresh, onError }: { resource: CourseResource; disabled: boolean; onRefresh: () => Promise<unknown>; onError: (message: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [title, setTitle] = useState(resource.title)
  const [resourceType, setResourceType] = useState<ResourceType>(resource.resource_type)
  const [location, setLocation] = useState(resource.resource_type === 'local_file' ? resource.local_path : resource.url)
  const [note, setNote] = useState(resource.note)
  const local = resourceType === 'local_file'
  const save = async (event: FormEvent) => {
    event.preventDefault()
    try { await api.updateResource(resource.id, { title, resource_type: resourceType, note, url: local ? '' : location, local_path: local ? location : '' }); setEditing(false); await onRefresh() } catch (requestError) { onError(errorMessage(requestError)) }
  }
  const remove = async () => {
    try { await api.deleteResource(resource.id); await onRefresh() } catch (requestError) { onError(errorMessage(requestError)) }
  }
  if (editing) return <form className="resource-row resource-row-edit" onSubmit={save}><div className="resource-form-grid"><label>Название<input required value={title} onChange={(event) => setTitle(event.target.value)} /></label><label>Тип<select value={resourceType} onChange={(event) => { const value = event.target.value as ResourceType; setResourceType(value); setLocation('') }}>{resourceTypes.map((value) => <option key={value} value={value}>{resourceTypeLabels[value]}</option>)}</select></label></div><label>{local ? 'Путь к файлу' : 'URL'}<input required type={local ? 'text' : 'url'} value={location} onChange={(event) => setLocation(event.target.value)} /></label><label>Заметка<textarea rows={2} value={note} onChange={(event) => setNote(event.target.value)} /></label><div className="form-actions"><button className="button secondary small" type="button" onClick={() => setEditing(false)}>Отмена</button><button className="button primary small" type="submit">Сохранить</button></div></form>
  return <div className="resource-row"><span className={`resource-type resource-type-${resource.resource_type}`}><Icon name={resource.resource_type === 'local_file' ? 'journal' : resource.resource_type === 'repository' ? 'briefcase' : resource.resource_type === 'video' ? 'chart' : 'book'} size={14} /></span><span className="resource-copy"><a href={courseResourceHref(resource)} target={resource.resource_type === 'local_file' ? undefined : '_blank'} rel={resource.resource_type === 'local_file' ? undefined : 'noreferrer'}>{resource.title}</a><small>{resource.resource_type === 'local_file' ? resource.local_path : resource.url}{resource.note ? ` · ${resource.note}` : ''}</small></span><button className="text-button" type="button" disabled={disabled} onClick={() => setEditing(true)}>Изменить</button><button className="danger-button" type="button" disabled={disabled} onClick={remove}>Удалить</button></div>
}

function RoadmapLinkForm({ module, options, onCancel, onDone, onError }: { module: CourseModule; options: RoadmapOption[]; onCancel: () => void; onDone: () => Promise<void>; onError: (message: string) => void }) {
  const linked = new Set(module.roadmap_links.map((link) => link.node_id))
  const available = options.filter(({ node }) => !linked.has(node.id))
  const [nodeId, setNodeId] = useState(available[0]?.node.id.toString() ?? '')
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!nodeId) return
    try { await api.createRoadmapLink(module.id, Number(nodeId)); await onDone() } catch (requestError) { onError(errorMessage(requestError)) }
  }
  return <form className="roadmap-link-form" onSubmit={submit}><label>Тема roadmap<select required value={nodeId} onChange={(event) => setNodeId(event.target.value)}><option value="" disabled>Выберите тему</option>{available.map(({ direction, node }) => <option key={node.id} value={node.id}>{direction.title} · {node.title}</option>)}</select></label><div className="form-actions"><button className="button secondary small" type="button" onClick={onCancel}>Отмена</button><button className="button primary small" type="submit" disabled={!nodeId}>Связать</button></div></form>
}

function CourseImportPanel({ onClose, onImported }: { onClose: () => void; onImported: (preview: CourseImportPreview) => Promise<void> }) {
  const [format, setFormat] = useState<'markdown' | 'json'>('markdown')
  const [content, setContent] = useState(markdownExample)
  const [preview, setPreview] = useState<CourseImportPreview | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const example = format === 'markdown' ? markdownExample : jsonExample
  const previewImport = async (event: FormEvent) => {
    event.preventDefault()
    setLoading(true)
    setError('')
    try { setPreview(await api.previewCourseImport(format, content)) } catch (requestError) { setError(errorMessage(requestError)); setPreview(null) } finally { setLoading(false) }
  }
  return <section className="course-import-panel" aria-labelledby="course-import-title"><div className="course-import-heading"><div><p className="eyebrow">Безопасный импорт</p><h2 id="course-import-title">Сначала preview, потом запись</h2><p>Текст проверяется и показывается до создания курса. Preview действует 15 минут и используется один раз.</p></div><button className="icon-button" type="button" aria-label="Закрыть импорт" onClick={onClose}><Icon name="close" size={16} /></button></div><form onSubmit={previewImport}><div className="import-toolbar"><label>Формат<select value={format} onChange={(event) => { const next = event.target.value as 'markdown' | 'json'; setFormat(next); setContent(next === 'markdown' ? markdownExample : jsonExample); setPreview(null) }}><option value="markdown">Markdown</option><option value="json">JSON</option></select></label><button className="text-button" type="button" onClick={() => setContent(example)}>Подставить пример</button></div><label>Содержимое<textarea className="import-textarea" rows={13} value={content} onChange={(event) => { setContent(event.target.value); setPreview(null) }} /></label>{error && <p className="form-error">{error}</p>}<div className="form-actions"><button className="button primary" type="submit" disabled={loading}>{loading ? 'Проверяем…' : 'Показать preview'}</button></div></form>{preview && <ImportPreview preview={preview} onImported={onImported} />}</section>
}

function ImportPreview({ preview, onImported }: { preview: CourseImportPreview; onImported: (preview: CourseImportPreview) => Promise<void> }) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const importNow = async () => {
    setLoading(true)
    setError('')
    try { await onImported(preview) } catch (requestError) { setError(errorMessage(requestError)) } finally { setLoading(false) }
  }
  return <div className="import-preview"><div className="import-preview-top"><div><span className="preview-badge">Preview готов</span><h3>{preview.course.title}</h3><p>{preview.course.description || 'Без описания'}{preview.course.provider ? ` · ${preview.course.provider}` : ''}</p></div><button className="button primary small" type="button" disabled={loading} onClick={importNow}>{loading ? 'Импортируем…' : 'Импортировать'}</button></div><div className="preview-summary"><span><strong>{preview.summary.modules}</strong> модулей</span><span><strong>{preview.summary.resources}</strong> материалов</span><span>до {new Date(preview.expires_at).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}</span></div><div className="preview-modules">{preview.course.modules.map((module) => <div key={module.title} className="preview-module"><strong>{module.title}</strong><small>{module.resources.length} материалов{module.description ? ` · ${module.description}` : ''}</small></div>)}</div>{error && <p className="form-error">{error}</p>}</div>
}
