import { request } from '../../shared/api/client'
import type { Settings, UpdateSettings } from '../../shared/types'

export const settingsApi = {
  settings: (signal?: AbortSignal) => request<Settings>('/settings', { signal }),
  updateSettings: (data: UpdateSettings) =>
    request<Settings>('/settings', { method: 'PUT', body: JSON.stringify(data) }),
}
