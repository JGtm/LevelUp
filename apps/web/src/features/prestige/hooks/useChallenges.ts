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
