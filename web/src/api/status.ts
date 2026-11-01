import {getJson} from './client'

// These types mirror internal/status/status.go on the backend.
// If you change a JSON field there, change it here too.

export type State = 'ok' | 'degraded' | 'down'

export interface ComponentStatus {
  name: string
  status: State
  latency_ms: number
  error?: string
  details?: Record<string, string | number | boolean>
}

export interface ServiceInfo {
  name: string
  api_version: string
  version: string
  commit: string
  build_time: string
  go_version: string
  env: string
  started_at: string
  uptime_seconds: number
}

export interface StatusReport {
  status: State
  service: ServiceInfo
  components: ComponentStatus[]
  checked_at: string
}

export function fetchStatus(token?: string, signal?: AbortSignal): Promise<StatusReport> {
  return getJson<StatusReport>('/api/v1/status', token, signal)
}
