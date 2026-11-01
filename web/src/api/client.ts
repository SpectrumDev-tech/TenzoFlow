// A tiny typed wrapper around fetch(). All requests to the backend go through here.

// Empty in development: requests go to the Vite dev server, which proxies /api to the backend.
// In production set VITE_API_BASE_URL only if the API lives on a different domain than the frontend.
const baseUrl: string = import.meta.env.VITE_API_BASE_URL ?? ''

/** The error format of the backend: {"error": {"code": "...", "message": "..."}} */
interface ErrorBody {
  error?: { code?: string; message?: string }
}

export class ApiError extends Error {
  /** HTTP status, or undefined when the server could not be reached at all. */
  readonly status: number | undefined
  readonly code: string | undefined

  constructor(message: string, status?: number, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export async function getJson<T>(path: string, token?: string, signal?: AbortSignal): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  let res: Response
  try {
    res = await fetch(`${baseUrl}${path}`, { headers, signal })
  } catch (e) {
    // An aborted request is not an error to show: the caller cancelled it on purpose.
    if (e instanceof DOMException && e.name === 'AbortError') {
      throw e
    }
    // fetch() rejects only when the network request itself failed: server down,
    // connection refused, CORS blocked, no internet. We cannot tell these apart from JS.
    throw new ApiError('Backend is unreachable', undefined, 'network_error')
  }

  if (!res.ok) {
    let message = `Request failed with HTTP ${res.status}`
    let code: string | undefined
    try {
      const body = (await res.json()) as ErrorBody
      message = body.error?.message ?? message
      code = body.error?.code
    } catch {
      // The body was not JSON (for example an error page of a proxy): keep the generic message.
    }
    throw new ApiError(message, res.status, code)
  }

  return (await res.json()) as T
}
