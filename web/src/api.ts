import type {
  Course,
  CourseImportPreview,
  CourseModule,
  CourseModuleRoadmapLink,
  CourseResource,
  CourseSummary,
  CreateCourse,
  CreateCourseModule,
  CreateDirection,
  CreateResource,
  CreateRoadmapNode,
  Direction,
  RoadmapNode,
  UpdateCourse,
  UpdateCourseModule,
  UpdateResource,
  UpdateRoadmapNode,
} from './types'

interface ApiErrorBody {
  error?: { code?: string; message?: string; details?: Record<string, unknown> }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
  })

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ApiErrorBody
    throw new Error(body.error?.message || `Ошибка запроса (${response.status})`)
  }

  return response.status === 204 ? (undefined as T) : ((await response.json()) as T)
}

export const api = {
  directions: (signal?: AbortSignal) => request<Direction[]>('/directions', { signal }),
  createDirection: (data: CreateDirection) =>
    request<Direction>('/directions', { method: 'POST', body: JSON.stringify(data) }),
  roadmap: (directionId: number, signal?: AbortSignal) =>
    request<RoadmapNode[]>(`/directions/${directionId}/roadmap`, { signal }),
  createNode: (data: CreateRoadmapNode) =>
    request<RoadmapNode>('/roadmap/nodes', { method: 'POST', body: JSON.stringify(data) }),
  updateNode: (id: number, data: UpdateRoadmapNode) =>
    request<RoadmapNode>(`/roadmap/nodes/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteNode: (id: number) => request<void>(`/roadmap/nodes/${id}`, { method: 'DELETE' }),
  createDependency: (nodeId: number, dependsOnNodeId: number) =>
    request(`/roadmap/nodes/${nodeId}/dependencies`, {
      method: 'POST',
      body: JSON.stringify({ depends_on_node_id: dependsOnNodeId }),
    }),
  deleteDependency: (nodeId: number, dependencyId: number) =>
    request<void>(`/roadmap/nodes/${nodeId}/dependencies/${dependencyId}`, { method: 'DELETE' }),
  courses: (signal?: AbortSignal) => request<CourseSummary[]>('/courses', { signal }),
  course: (id: number, signal?: AbortSignal) => request<Course>(`/courses/${id}`, { signal }),
  createCourse: (data: CreateCourse) =>
    request<Course>('/courses', { method: 'POST', body: JSON.stringify(data) }),
  updateCourse: (id: number, data: UpdateCourse) =>
    request<Course>(`/courses/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  archiveCourse: (id: number) => request<void>(`/courses/${id}`, { method: 'DELETE' }),
  createCourseModule: (courseId: number, data: CreateCourseModule) =>
    request<CourseModule>(`/courses/${courseId}/modules`, { method: 'POST', body: JSON.stringify(data) }),
  updateCourseModule: (id: number, data: UpdateCourseModule) =>
    request<CourseModule>(`/course-modules/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteCourseModule: (id: number) => request<void>(`/course-modules/${id}`, { method: 'DELETE' }),
  createResource: (moduleId: number, data: CreateResource) =>
    request<CourseResource>(`/course-modules/${moduleId}/resources`, { method: 'POST', body: JSON.stringify(data) }),
  updateResource: (id: number, data: UpdateResource) =>
    request<CourseResource>(`/resources/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteResource: (id: number) => request<void>(`/resources/${id}`, { method: 'DELETE' }),
  createRoadmapLink: (moduleId: number, nodeId: number) =>
    request<CourseModuleRoadmapLink>(`/course-modules/${moduleId}/roadmap-links`, {
      method: 'POST', body: JSON.stringify({ node_id: nodeId }),
    }),
  deleteRoadmapLink: (moduleId: number, linkId: number) =>
    request<void>(`/course-modules/${moduleId}/roadmap-links/${linkId}`, { method: 'DELETE' }),
  previewCourseImport: (format: 'markdown' | 'json', content: string) =>
    request<CourseImportPreview>('/courses/import/preview', {
      method: 'POST', body: JSON.stringify({ format, content }),
    }),
  importCourse: (previewId: string) =>
    request<Course>('/courses/import', { method: 'POST', body: JSON.stringify({ preview_id: previewId }) }),
}
