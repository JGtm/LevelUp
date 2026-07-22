/**
 * Hooks React Query — défis Prestige.
 *
 * Référence : Phase 5 du plan IMPL_PRESTIGE.md.
 * Centralise les queries (cache + refetch) et mutations (CRUD défis).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  prestigeApi,
  type ChallengeStatus,
  type CreateChallengeBody,
  type UpdateChallengeBody,
} from '@/lib/prestige'
import { queryKeys } from '@/lib/query/keys'

// ─── Queries ───

export function useChallenges(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: queryKeys.challenge.list(userId, titleSlug),
    queryFn: () => prestigeApi.listActiveChallenges(userId, titleSlug),
    retry: false,
    enabled: !!userId && !!titleSlug,
  })
}

/** Statuts terminaux servis à l'historique Réalisations (défis passés). */
const TERMINAL_CHALLENGE_STATUSES: ChallengeStatus[] = [
  'completed',
  'expired',
  'abandoned',
  'archived',
]

/**
 * useChallengeHistory — défis terminaux (complétés/expirés/abandonnés/retirés)
 * pour la surface Historique de l'onglet Réalisations. Clé de cache distincte de
 * `useChallenges` (actifs).
 */
export function useChallengeHistory(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: queryKeys.challenge.history(userId, titleSlug),
    queryFn: () => prestigeApi.listChallenges(userId, titleSlug, TERMINAL_CHALLENGE_STATUSES),
    retry: false,
    enabled: !!userId && !!titleSlug,
  })
}

// ─── Mutations ───

export function useCreateChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateChallengeBody) => prestigeApi.createChallenge(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.challenge.list(userId, titleSlug) })
      qc.invalidateQueries({ queryKey: queryKeys.prestige.meAll(userId) })
    },
  })
}

export function useUpdateChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateChallengeBody }) =>
      prestigeApi.updateChallenge(id, body, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.challenge.list(userId, titleSlug) })
    },
  })
}

export function useAbandonChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => prestigeApi.abandonChallenge(id, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.challenge.list(userId, titleSlug) })
    },
  })
}

// ─── Mode pilote (auto-attribution, B3) ───

/**
 * usePilotMode — mutations enable/disable du mode pilote. L'état ON/OFF n'est
 * pas persisté côté serveur : il se dérive de la présence de défis
 * `mode === 'pilote'` actifs (cf. useChallenges). Les deux mutations invalident
 * donc la liste des défis + le total PP (l'auto-attribution crée des défis).
 */
export function usePilotMode(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: queryKeys.challenge.list(userId, titleSlug) })
    qc.invalidateQueries({ queryKey: queryKeys.prestige.meAll(userId) })
  }
  const enable = useMutation({
    mutationKey: queryKeys.prestige.pilotMode(userId, titleSlug),
    mutationFn: () => prestigeApi.enablePilotMode(userId, titleSlug),
    onSuccess: invalidate,
  })
  const disable = useMutation({
    mutationKey: queryKeys.prestige.pilotMode(userId, titleSlug),
    mutationFn: () => prestigeApi.disablePilotMode(userId, titleSlug),
    onSuccess: invalidate,
  })
  return { enable, disable }
}
