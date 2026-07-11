import type {
  CreateDirection,
  CreateRoadmapNode,
  Direction,
  RoadmapNode,
  UpdateRoadmapNode,
} from './types'

interface ApiErrorBody {
  error?: { message?: string }
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
}
