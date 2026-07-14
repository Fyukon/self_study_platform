export type RoadmapStatus = 'not_started' | 'learning' | 'practicing' | 'understood'
export type NodeType = 'section' | 'concept' | 'practice' | 'project' | 'checkpoint'

export interface Direction {
  id: number
  title: string
  description: string
  icon: string
  position: number
  is_archived: boolean
  created_at: string
  updated_at: string
}

export interface NodeDependency {
  id: number
  node_id: number
  depends_on_node_id: number
  created_at: string
}

export interface RoadmapNode {
  id: number
  direction_id: number
  parent_id: number | null
  title: string
  description: string
  node_type: NodeType
  status: RoadmapStatus
  needs_review: boolean
  confidence: number
  position: number
  next_action: string
  target_date: string | null
  last_reviewed_at: string | null
  estimated_hours: number | null
  created_at: string
  updated_at: string
  dependencies: NodeDependency[]
  children?: RoadmapNode[]
}

export type CreateDirection = Pick<Direction, 'title' | 'description'>

export type CreateRoadmapNode = Pick<RoadmapNode, 'direction_id' | 'title'> &
  Partial<
    Pick<
      RoadmapNode,
      | 'parent_id'
      | 'description'
      | 'node_type'
      | 'status'
      | 'needs_review'
      | 'confidence'
      | 'position'
      | 'next_action'
    >
  >

export type UpdateRoadmapNode = Partial<
  Pick<
    RoadmapNode,
    | 'title'
    | 'description'
    | 'node_type'
    | 'status'
    | 'needs_review'
    | 'confidence'
    | 'next_action'
    | 'last_reviewed_at'
  >
>

export type CourseModuleStatus = 'not_started' | 'in_progress' | 'completed'
export type ResourceType = 'book' | 'video' | 'repository' | 'article' | 'link' | 'local_file'

export interface CourseSummary {
  id: number
  title: string
  description: string
  provider: string
  source_url: string
  position: number
  is_archived: boolean
  module_count: number
  resource_count: number
  created_at: string
  updated_at: string
}

export interface CourseResource {
  id: number
  module_id: number
  title: string
  resource_type: ResourceType
  url: string
  local_path: string
  note: string
  position: number
  created_at: string
  updated_at: string
}

export interface CourseModuleRoadmapLink {
  id: number
  module_id: number
  node_id: number
  node_title: string
  direction_id: number
  direction_title: string
  created_at: string
}

export interface CourseModule {
  id: number
  course_id: number
  title: string
  description: string
  status: CourseModuleStatus
  position: number
  created_at: string
  updated_at: string
  resources: CourseResource[]
  roadmap_links: CourseModuleRoadmapLink[]
}

export interface Course {
  id: number
  title: string
  description: string
  provider: string
  source_url: string
  position: number
  is_archived: boolean
  created_at: string
  updated_at: string
  modules: CourseModule[]
}

export type CreateCourse = Pick<Course, 'title'> & Partial<Pick<Course, 'description' | 'provider' | 'source_url' | 'position'>>
export type UpdateCourse = Partial<Pick<Course, 'title' | 'description' | 'provider' | 'source_url' | 'position' | 'is_archived'>>
export type CreateCourseModule = Pick<CourseModule, 'title'> & Partial<Pick<CourseModule, 'description' | 'status' | 'position'>>
export type UpdateCourseModule = Partial<Pick<CourseModule, 'title' | 'description' | 'status' | 'position'>>
export type CreateResource = Pick<CourseResource, 'title' | 'resource_type'> & Partial<Pick<CourseResource, 'url' | 'local_path' | 'note' | 'position'>>
export type UpdateResource = Partial<Pick<CourseResource, 'title' | 'resource_type' | 'url' | 'local_path' | 'note' | 'position'>>

export interface CourseImportResource {
  title: string
  resource_type: ResourceType
  url: string
  local_path: string
  note: string
}

export interface CourseImportModule {
  title: string
  description: string
  resources: CourseImportResource[]
}

export interface CourseImportDocument {
  title: string
  description: string
  provider: string
  source_url: string
  modules: CourseImportModule[]
}

export interface CourseImportPreview {
  preview_id: string
  format: 'markdown' | 'json'
  course: CourseImportDocument
  summary: { modules: number; resources: number }
  expires_at: string
}
