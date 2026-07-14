import { request } from '../../shared/api/client'
import type {
  Course,
  CourseImportPreview,
  CourseModule,
  CourseModuleRoadmapLink,
  CourseResource,
  CourseSummary,
  CreateCourse,
  CreateCourseModule,
  CreateResource,
  UpdateCourse,
  UpdateCourseModule,
  UpdateResource,
} from '../../shared/types'

export const coursesApi = {
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
