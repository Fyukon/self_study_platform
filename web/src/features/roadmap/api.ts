import { request } from '../../shared/api/client'
import type {
  CreateDirection,
  CreateRoadmapNode,
  Direction,
  RoadmapNode,
  UpdateRoadmapNode,
} from '../../shared/types'

export const roadmapApi = {
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
}
