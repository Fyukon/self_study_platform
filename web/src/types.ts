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
