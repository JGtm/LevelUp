/**
 * Hooks React Query pour le module Prestige.
 *
 * Centralise les queries (cache + refetch) et mutations (CRUD défis, arcs,
 * squad challenges). Les invalidations sont déclenchées sur les bonnes
 * cache keys pour garder l'UI synchrone après chaque action.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  prestigeApi,
  type CreateChallengeBody,
  type UpdateChallengeBody,
  type CreateArcBody,
  type Tier,
} from '@/lib/prestige'

// ─── Cache keys ───

const keys = {
  challenges: (userId: string, titleSlug: string) =>
    ['prestige', 'challenges', userId, titleSlug] as const,
  challenge: (id: string) => ['prestige', 'challenge', id] as const,
  arcs: (userId: string, titleSlug: string) =>
    ['prestige', 'arcs', userId, titleSlug] as const,
  me: (userId: string, titleSlug?: string) =>
    ['prestige', 'me', userId, titleSlug] as const,
  templates: (userId: string, titleSlug: string) =>
    ['prestige', 'templates', userId, titleSlug] as const,
}

// ─── Queries ───

export function useChallenges(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: keys.challenges(userId, titleSlug),
    queryFn: () => prestigeApi.listActiveChallenges(userId, titleSlug),
    retry: false,
    enabled: !!userId && !!titleSlug,
  })
}

export function useArcs(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: keys.arcs(userId, titleSlug),
    queryFn: () => prestigeApi.listArcs(userId, titleSlug),
    retry: false,
    enabled: !!userId && !!titleSlug,
  })
}

export function useMyPrestige(userId: string, titleSlug?: string) {
  return useQuery({
    queryKey: keys.me(userId, titleSlug),
    queryFn: () => prestigeApi.getMyPrestige(userId, titleSlug),
    retry: false,
    enabled: !!userId,
  })
}

export function useSuggestedTemplates(userId: string, titleSlug: string, count = 3) {
  return useQuery({
    queryKey: keys.templates(userId, titleSlug),
    queryFn: () => prestigeApi.suggestTemplates(userId, titleSlug, count),
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
      qc.invalidateQueries({ queryKey: keys.challenges(userId, titleSlug) })
      qc.invalidateQueries({ queryKey: keys.me(userId, titleSlug) })
    },
  })
}

export function useUpdateChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: UpdateChallengeBody }) =>
      prestigeApi.updateChallenge(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.challenges(userId, titleSlug) })
    },
  })
}

export function useAbandonChallenge(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => prestigeApi.abandonChallenge(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.challenges(userId, titleSlug) })
    },
  })
}

export function useCreateArc(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateArcBody) => prestigeApi.createArc(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.arcs(userId, titleSlug) })
    },
  })
}

export function useJoinSquadChallenge() {
  return useMutation({
    mutationFn: ({
      challengeId,
      userId,
      chosenTier,
      isPrivate,
    }: {
      challengeId: string
      userId: string
      chosenTier?: Tier
      isPrivate?: boolean
    }) =>
      prestigeApi.joinSquadChallenge(challengeId, {
        user_id: userId,
        chosen_tier: chosenTier,
        is_private: isPrivate,
      }),
  })
}
