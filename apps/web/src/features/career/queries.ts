/**
 * Queries TanStack Query — page Carrière (Slice 2).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  CareerPageResponse,
  CareerTopMatchesResponse,
  CareerEncountersResponse,
  CareerHighlightMatchesResponse,
  CareerHighlightFilters,
  CareerTopEncountersResponse,
  CareerRivalsResponse,
  CareerCSRResponse,
} from '@/lib/api/types'

export function useCareerPage(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.career(playerSlug),
    queryFn: () => api.get<CareerPageResponse>(`/players/${playerSlug}/pages/career`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

// V8b — top matches et encounters ne sont PAS servis par /pages/career : ils ont
// leurs endpoints dédiés, fetch d'entrée de page (le contrat renvoie best_matches /
// worst_matches et teammates / enemies, jamais un champ preview).
export function useCareerTopMatches(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.careerTopMatches(playerSlug),
    queryFn: () => api.get<CareerTopMatchesResponse>(`/players/${playerSlug}/pages/career/top-matches`),
    enabled: !!playerSlug,
    staleTime: 10 * 60 * 1000,
  })
}

export function useCareerEncounters(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.careerEncounters(playerSlug),
    queryFn: () => api.get<CareerEncountersResponse>(`/players/${playerSlug}/pages/career/encounters`),
    enabled: !!playerSlug,
    staleTime: 10 * 60 * 1000,
  })
}

// Section "Matchs marquants" — 15 best + 15 worst au format ExplorerMatchRow,
// filtrés par expérience (ranked/unranked) et saisons (multi-select).
// La réponse inclut les cascade counts (available_experience, available_seasons)
// pour alimenter les dropdowns.
export function useCareerHighlightMatches(playerSlug: string, filters: CareerHighlightFilters = {}) {
  const params = buildHighlightFilterParams(filters)
  return useQuery({
    queryKey: queryKeys.careerHighlightMatches(playerSlug, params),
    queryFn: () => {
      const qs = params ? `?${params}` : ''
      return api.get<CareerHighlightMatchesResponse>(
        `/players/${playerSlug}/pages/career/highlight-matches${qs}`,
      )
    },
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

// buildHighlightFilterParams projette les filtres en query string canonique.
// Retourne "" si aucun filtre (skip query string complet pour cache hit
// optimal sur l'état "tous"). Tri pour stabilité de la query key.
function buildHighlightFilterParams(filters: CareerHighlightFilters): string {
  const params = new URLSearchParams()
  if (filters.experience && filters.experience !== 'all') {
    params.set('experience', filters.experience)
  }
  if (filters.season_ids && filters.season_ids.length > 0) {
    params.set('season_ids', [...filters.season_ids].sort().join(','))
  }
  if (filters.mode_uis && filters.mode_uis.length > 0) {
    params.set('mode_uis', [...filters.mode_uis].sort().join(','))
  }
  if (filters.playlist_names && filters.playlist_names.length > 0) {
    params.set('playlist_names', [...filters.playlist_names].sort().join(','))
  }
  return params.toString()
}

// Section "Joueurs les plus croisés (hors amis)" — 10 lignes MatchEncounterRow.
export function useCareerTopEncounters(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.careerTopEncounters(playerSlug),
    queryFn: () =>
      api.get<CareerTopEncountersResponse>(`/players/${playerSlug}/pages/career/top-encounters`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

// Section "Top némésis" / "Top souffre-douleur" — 10 chacun.
export function useCareerRivals(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.careerRivals(playerSlug),
    queryFn: () => api.get<CareerRivalsResponse>(`/players/${playerSlug}/pages/career/rivals`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

export function useCareerCSRs(playerSlug: string, season?: string) {
  return useQuery({
    queryKey: queryKeys.careerCSRs(playerSlug, season),
    queryFn: () =>
      api.get<CareerCSRResponse>(
        `/players/${playerSlug}/pages/career/csrs${season ? `?season=${encodeURIComponent(season)}` : ''}`,
      ),
    enabled: !!playerSlug,
    staleTime: 10 * 60 * 1000,
  })
}
