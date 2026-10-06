export type FieldErrors = Record<string, string>

/** Error thrown for any failed API call, carrying the backend error contract. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  /** Per-field messages from a 422 response, keyed by JSON field name. */
  readonly fields: FieldErrors

  constructor(status: number, code: string, message: string, fields: FieldErrors = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }

  get isValidation(): boolean {
    return this.status === 422
  }
}

interface ErrorBody {
  error: string
  message?: string
  fields?: FieldErrors
}

function isErrorBody(value: unknown): value is ErrorBody {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as { error?: unknown }).error === 'string'
  )
}

function fallbackMessage(status: number, code: string): string {
  if (status === 422) return 'Please fix the highlighted fields.'
  if (status === 401) return 'Please log in again.'
  if (status === 403) return 'You do not have permission to do that.'
  if (status === 404) return 'The entity was not found. It may have been deleted.'
  // The Vite dev proxy answers 502/503/504 when the backend is not running.
  if (status >= 502 && status <= 504) return 'Cannot reach the server. Is the backend running?'
  if (status >= 500) return 'Something went wrong on the server. Please try again.'
  return `Request failed (${code}).`
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  // FormData (file uploads) is sent as is: the browser sets the multipart
  // Content-Type with its boundary. Anything else is sent as JSON.
  const isForm = body instanceof FormData
  let res: Response
  try {
    res = await fetch(`/api${path}`, {
      method,
      headers: {
        Accept: 'application/json',
        ...(body !== undefined && !isForm && { 'Content-Type': 'application/json' }),
      },
      body: isForm ? body : body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Cannot reach the server. Is the backend running?')
  }

  if (res.status === 204) {
    return undefined as T
  }

  const payload: unknown = await res.json().catch(() => null)

  if (!res.ok) {
    if (isErrorBody(payload)) {
      throw new ApiError(
        res.status,
        payload.error,
        payload.message ?? fallbackMessage(res.status, payload.error),
        payload.fields,
      )
    }
    throw new ApiError(res.status, 'http_error', fallbackMessage(res.status, 'http_error'))
  }

  return (payload as { data: T }).data
}

export const http = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body: unknown) => request<T>('PATCH', path, body),
  delete: (path: string) => request<void>('DELETE', path),
}
