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

/**
 * Extrait un message lisible d'une erreur catchée par TanStack Query ou un
 * onError. `ApiError` est un objet plain (pas `instanceof Error`), donc le
 * pattern usuel `err instanceof Error ? err.message : undefined` perd toujours
 * le message backend. Cette helper accepte `ApiError`, `Error` et `unknown`.
 */
export function apiErrorMessage(err: unknown): string | undefined {
  if (!err) return undefined
  if (typeof err === 'object' && err !== null && 'message' in err) {
    const m = (err as { message?: unknown }).message
    if (typeof m === 'string' && m.length > 0) return m
  }
  if (err instanceof Error) return err.message
  return undefined
}

/**
 * Extrait le code machine d'une `ApiError` (ex: "match_not_participant",
 * "player_forbidden", "session_not_found"). Permet aux pages de distinguer les
 * cas d'accès (ADR 0029) sans matcher fragilement sur le message HTTP.
 */
export function apiErrorCode(err: unknown): string | undefined {
  if (err && typeof err === 'object' && 'code' in err) {
    const c = (err as { code?: unknown }).code
    if (typeof c === 'string' && c.length > 0) return c
  }
  return undefined
}

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

/** Base URL pour navigation plein-page (OAuth redirects). Exposée pour XboxLoginPage. */
export const API_BASE_URL = BASE_URL

/**
 * Sprint 44 : titre courant pour les requêtes API.
 * Mis à jour par le store appShell lors du bootstrap et des switchs de titre.
 * Si "halo_infinite" (défaut), le header n'est pas envoyé (rétrocompatibilité).
 */
let _currentTitleSlug = 'halo_infinite'

/** Appelé par le store pour mettre à jour le titre courant. */
export function setApiTitleSlug(slug: string): void {
  _currentTitleSlug = slug
}

/**
 * Titre courant tel que vu par le client API. Exposé pour que les stores de
 * filtres estampillent le titre actif dans le deep-link `?f=` (share-link), afin
 * qu'un filtre ne se réapplique qu'au titre pour lequel il a été généré.
 */
export function getApiTitleSlug(): string {
  return _currentTitleSlug
}

function getTitleHeader(): Record<string, string> {
  if (_currentTitleSlug && _currentTitleSlug !== 'halo_infinite') {
    return { 'X-LevelUp-Title': _currentTitleSlug }
  }
  return {}
}

/**
 * Locale courante pour les réponses API. Mis à jour par appShellStore.
 * Le backend lit ce header en priorité (fallback sur app_settings.lang) pour
 * sélectionner les labels FR/EN dans les payloads (map names, mode names…).
 */
let _currentLocale: 'fr' | 'en' = 'fr'

/** Appelé par le store pour mettre à jour la locale courante. */
export function setApiLocale(locale: 'fr' | 'en'): void {
  _currentLocale = locale
}

function getLocaleHeader(): Record<string, string> {
  return { 'X-LevelUp-Locale': _currentLocale }
}

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
      ...getTitleHeader(),
      ...getLocaleHeader(),
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
    // Session expirée en cours de navigation → signal au RootLayout
    if (response.status === 401 && errorBody.code === 'auth_required' && !path.includes('/bootstrap')) {
      window.dispatchEvent(new CustomEvent('levelup:auth-required'))
    }
    throw err
  }

  if (response.status === 204) {
    return undefined as unknown as T
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

  /**
   * Envoie un FormData en multipart/form-data.
   * Ne pose PAS Content-Type : le browser calcule automatiquement le boundary.
   */
  postForm: async <T>(path: string, form: FormData): Promise<T> => {
    const url = `${BASE_URL}${path}`
    const response = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        ...getTitleHeader(),
        ...getLocaleHeader(),
        // Content-Type intentionnellement absent → boundary auto
      },
      body: form,
    })

    if (!response.ok) {
      let errorBody: Partial<ApiError> = {}
      try {
        errorBody = await response.json()
      } catch { /* body non-JSON */ }
      const err: ApiError = {
        code: errorBody.code ?? 'upload_error',
        message: errorBody.message ?? `Erreur HTTP ${response.status}`,
        retryable: errorBody.retryable ?? response.status >= 500,
        details: errorBody.details,
        field_errors: errorBody.field_errors,
        status: response.status,
      }
      throw err
    }

    return response.json() as Promise<T>
  },
}
