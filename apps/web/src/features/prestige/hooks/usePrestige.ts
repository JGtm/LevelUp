/**
 * Hooks React Query — Prestige (PP, niveau, templates, squad).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { prestigeApi, type Tier } from '@/lib/prestige'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

export function useMyPrestige(userId: string, titleSlug?: string) {
  return useQuery({
    queryKey: queryKeys.prestige.me(userId, titleSlug),
    queryFn: () => prestigeApi.getMyPrestige(userId, titleSlug),
    retry: false,
    enabled: !!userId,
  })
}

export function useSuggestedTemplates(userId: string, titleSlug: string, count = 3) {
  const locale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.prestige.templates(userId, titleSlug, locale),
    queryFn: () => prestigeApi.suggestTemplates(userId, titleSlug, count),
    retry: false,
    enabled: !!userId && !!titleSlug,
  })
}

export function useJoinSquadChallenge() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      challengeId,
      userId,
      chosenTier,
      isPrivate,
    }: {
      challengeId: string
      userId: string
      // squadId n'est pas envoyé au backend (le défi porte déjà son squad) : il
      // sert uniquement à cibler l'invalidation du cache défis au succès.
      squadId: string
      chosenTier?: Tier
      isPrivate?: boolean
    }) =>
      prestigeApi.joinSquadChallenge(challengeId, {
        user_id: userId,
        chosen_tier: chosenTier,
        is_private: isPrivate,
      }),
    onSuccess: (_data, { squadId }) => {
      qc.invalidateQueries({ queryKey: queryKeys.squad.challengesAll(squadId) })
    },
  })
}
