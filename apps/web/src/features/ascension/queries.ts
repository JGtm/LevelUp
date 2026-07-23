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
  ActivityCalendar,
  MilestonesResponse,
  PatternReport,
  PlayerProfile,
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

/** Profil de jeu complet (sections A1/A2/B/C). */
export function useProfile(playerSlug: string, windowDays = 30, enabled = true) {
  return useQuery<PlayerProfile>({
    queryKey: queryKeys.progressionProfile(playerSlug, windowDays),
    queryFn: () =>
      api.get<PlayerProfile>(`/players/${playerSlug}/profile?window_days=${windowDays}`),
    enabled: !!playerSlug && enabled,
    refetchInterval: 120_000,
    refetchOnWindowFocus: false,
    staleTime: 60_000,
  })
}

/** Rapport patterns contextuels + comportementaux + leviers. */
export function usePatterns(playerSlug: string, n = 50, enabled = true) {
  return useQuery<PatternReport>({
    queryKey: queryKeys.progressionPatterns(playerSlug, n),
    queryFn: () =>
      api.get<PatternReport>(`/players/${playerSlug}/patterns?n=${n}`),
    enabled: !!playerSlug && enabled,
    refetchInterval: 120_000,
    refetchOnWindowFocus: false,
    staleTime: 60_000,
  })
}

/** Calendrier d'activité (jours joués sur `days` jours, défaut 90 — DEC-5/D3). */
export function useActivityCalendar(playerSlug: string, days = 90, enabled = true) {
  return useQuery<ActivityCalendar>({
    queryKey: queryKeys.progressionActivity(playerSlug, days),
    queryFn: () =>
      api.get<ActivityCalendar>(`/players/${playerSlug}/activity-calendar?days=${days}`),
    enabled: !!playerSlug && enabled,
    refetchInterval: 120_000,
    refetchOnWindowFocus: false,
    staleTime: 60_000,
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
