/**
 * Client HTTP centralisé — toutes les requêtes vers /api/v1 passent ici.
 *
 * - Base URL configurable via VITE_API_BASE_URL (défaut /api/v1)
 * - Headers communs : Content-Type, Accept
 * - Cookies : credentials: "include" pour la session httpOnly
 * - Erreurs HTTP : transformées en ApiError lisible
 */

export interface ApiError {
  code: string
  message: string
  retryable: boolean
  details?: unknown
  field_errors?: FieldError[]
  status: number
}

export interface FieldError {
  field: string
  message: string
  code?: string
}

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

async function request<T>(
  method: string,
  path: string,
  options?: {
    body?: unknown
    headers?: Record<string, string>
  },
): Promise<T> {
  const url = `${BASE_URL}${path}`
  const response = await fetch(url, {
    method,
    credentials: 'include', // cookies httpOnly (session)
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...options?.headers,
    },
    body: options?.body != null ? JSON.stringify(options.body) : undefined,
  })

  if (!response.ok) {
    let errorBody: Partial<ApiError> = {}
    try {
      errorBody = await response.json()
    } catch {
      // body non-JSON, on laisse errorBody vide
    }
    const err: ApiError = {
      code: errorBody.code ?? 'unknown_error',
      message: errorBody.message ?? `Erreur HTTP ${response.status}`,
      retryable: errorBody.retryable ?? response.status >= 500,
      details: errorBody.details,
      field_errors: errorBody.field_errors,
      status: response.status,
    }
    throw err
  }

  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string, headers?: Record<string, string>) =>
    request<T>('GET', path, { headers }),

  post: <T>(path: string, body?: unknown, headers?: Record<string, string>) =>
    request<T>('POST', path, { body, headers }),

  patch: <T>(path: string, body?: unknown, headers?: Record<string, string>) =>
    request<T>('PATCH', path, { body, headers }),

  put: <T>(path: string, body?: unknown, headers?: Record<string, string>) =>
    request<T>('PUT', path, { body, headers }),

  delete: <T>(path: string, headers?: Record<string, string>) =>
    request<T>('DELETE', path, { headers }),
}
