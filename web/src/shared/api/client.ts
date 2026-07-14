export interface ApiErrorBody {
  error?: { code?: string; message?: string; details?: Record<string, unknown> }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
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
