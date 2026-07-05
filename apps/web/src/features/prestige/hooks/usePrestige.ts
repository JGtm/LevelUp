/**
 * Hooks React Query — Prestige (PP, niveau, templates, squad).
 */
import { useQuery, useMutation } from '@tanstack/react-query'
import { prestigeApi, type Tier } from '@/lib/prestige'
import { queryKeys } from '@/lib/query/keys'

export function useMyPrestige(userId: string, titleSlug?: string) {
  return useQuery({
    queryKey: queryKeys.prestige.me(userId, titleSlug),
    queryFn: () => prestigeApi.getMyPrestige(userId, titleSlug),
    retry: false,
    enabled: !!userId,
  })
}

export function useSuggestedTemplates(userId: string, titleSlug: string, count = 3) {
  return useQuery({
    queryKey: queryKeys.prestige.templates(userId, titleSlug),
    queryFn: () => prestigeApi.suggestTemplates(userId, titleSlug, count),
    retry: false,
    enabled: !!userId && !!titleSlug,
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
