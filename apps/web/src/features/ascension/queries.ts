/**
 * TanStack Query hooks pour la couche progression V2 (Ascension).
 *
 * Polling : modéré (60s sur badge + dashboard). Refetch on focus actif.
 * Cf. queryKeys.progressionStreaks/Records/Milestones dans @/lib/query/keys.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MilestonesResponse,
  RecordsResponse,
  StreaksResponse,
} from './types'

/** Liste des streaks (active + historique). */
export function useStreaks(playerSlug: string, enabled = true) {
  return useQuery<StreaksResponse>({
    queryKey: queryKeys.progressionStreaks(playerSlug),
    queryFn: () =>
      api.get<StreaksResponse>(`/players/${playerSlug}/streaks`),
    enabled: !!playerSlug && enabled,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
    staleTime: 30_000,
  })
}

/**
 * Records (PB + timeline). Accepte un `historyLimit` optionnel (1-200,
 * défaut 50 côté serveur).
 */
export function useRecords(
  playerSlug: string,
  options?: { historyLimit?: number; enabled?: boolean },
) {
  const limit = options?.historyLimit
  const qs = limit ? `?history_limit=${limit}` : ''
  return useQuery<RecordsResponse>({
    queryKey: queryKeys.progressionRecords(playerSlug, limit),
    queryFn: () =>
      api.get<RecordsResponse>(`/players/${playerSlug}/records${qs}`),
    enabled: !!playerSlug && (options?.enabled ?? true),
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
    staleTime: 30_000,
  })
}

/** Catalogue de milestones avec statut Earned joint serveur. */
export function useMilestones(playerSlug: string, enabled = true) {
  return useQuery<MilestonesResponse>({
    queryKey: queryKeys.progressionMilestones(playerSlug),
    queryFn: () =>
      api.get<MilestonesResponse>(`/players/${playerSlug}/milestones`),
    enabled: !!playerSlug && enabled,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
    staleTime: 30_000,
  })
}
