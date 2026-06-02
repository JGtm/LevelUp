/**
 * Hooks React Query — défis Prestige.
 *
 * Référence : Phase 5 du plan IMPL_PRESTIGE.md.
 * Centralise les queries (cache + refetch) et mutations (CRUD défis).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  prestigeApi,
  type CreateChallengeBody,
  type UpdateChallengeBody,
} from '@/lib/prestige'
import { prestigeKeys } from './usePrestige'

// ─── Cache keys ───

export const challengeKeys = {
  list: (userId: string, titleSlug: string) =>
    ['prestige', 'challenges', userId, titleSlug] as const,
  one: (id: string) => ['prestige', 'challenge', id] as const,
}

// ─── Queries ───

export function useChallenges(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: challengeKeys.list(userId, titleSlug),
    queryFn: () => prestigeApi.listActiveChallenges(userId, titleSlug),
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
      qc.invalidateQueries({ queryKey: challengeKeys.list(userId, titleSlug) })
      qc.invalidateQueries({ queryKey: prestigeKeys.meAll(userId) })
    },
  })
}

export function useUpdateChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateChallengeBody }) =>
      prestigeApi.updateChallenge(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: challengeKeys.list(userId, titleSlug) })
    },
  })
}

export function useAbandonChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => prestigeApi.abandonChallenge(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: challengeKeys.list(userId, titleSlug) })
    },
  })
}
