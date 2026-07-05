/**
 * Queries pour le drawer feedback.
 *
 * 🔴 SÉCURITÉ — fetch direct vers api.github.com
 *
 * Le wrapper [`api`](@/lib/api/client) du projet a `credentials: 'include'`
 * + injecte le header `X-LevelUp-Title` quand le slug ≠ `halo_infinite`.
 * Le ré-utiliser pour appeler `api.github.com` leakerait ce header et
 * tenterait d'envoyer les cookies de session LevelUp à GitHub.
 *
 * → On utilise donc `fetch()` direct avec `credentials: 'omit'` et un header
 *   `Accept` minimal. **Ne jamais switcher vers `api.get(...)` ici.**
 *
 * 🔄 FUTUR — repo privé
 * Si demain le repo passe privé, la GitHub Search API publique retournera
 * 404 sans token auth → la section "Issues similaires" sera masquée
 * silencieusement. Pour migrer : exposer un endpoint backend
 * `GET /api/v1/feedback/search-issues?q=...` qui proxy l'appel via un PAT
 * readonly stocké côté Go API.
 */
import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { queryKeys } from '@/lib/query/keys'
import { buildSearchIssuesUrl } from './buildIssueUrl'
import { log } from './_logger'

interface GitHubIssue {
  number: number
  title: string
  html_url: string
  state: string
}

interface GitHubSearchResponse {
  items: GitHubIssue[]
  total_count: number
}

export interface SimilarIssueRef {
  number: number
  title: string
  url: string
}

const MIN_QUERY_LENGTH = 3
const DEBOUNCE_MS = 500
const STALE_MS = 60_000

/**
 * Hook : retourne jusqu'à 3 issues GitHub similaires au titre saisi.
 * Debounce 500 ms côté state, query react-query gated par `enabled`.
 */
export function useSimilarIssues(rawTitle: string, enabled: boolean) {
  const debounced = useDebounced(rawTitle, DEBOUNCE_MS)
  const trimmed = debounced.trim()
  const queryEnabled = enabled && trimmed.length >= MIN_QUERY_LENGTH

  return useQuery<SimilarIssueRef[]>({
    queryKey: queryKeys.feedbackSimilarIssues(trimmed),
    queryFn: () => fetchSimilarIssues(trimmed),
    enabled: queryEnabled,
    staleTime: STALE_MS,
    retry: false,
  })
}

async function fetchSimilarIssues(title: string): Promise<SimilarIssueRef[]> {
  const url = buildSearchIssuesUrl(title)
  let response: Response
  try {
    response = await fetch(url, {
      credentials: 'omit',
      headers: { Accept: 'application/vnd.github+json' },
    })
  } catch {
    log.warn('similar:fetch_failed', 'GitHub search API unreachable')
    return []
  }
  if (response.status === 403) {
    log.warn('similar:rate_limited', 'GitHub search API rate-limited')
    return []
  }
  if (!response.ok) {
    log.warn('similar:fetch_failed', `GitHub search returned ${response.status}`)
    return []
  }
  const data = (await response.json()) as GitHubSearchResponse
  return data.items.slice(0, 3).map((it) => ({
    number: it.number,
    title: it.title,
    url: it.html_url,
  }))
}

function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])
  return debounced
}
