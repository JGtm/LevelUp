/**
 * Hooks React Query — PlayerProfile V1 (Ascension §4-§5).
 *
 * - usePlayerProfile : profil complet (stale 5min)
 * - useActiveCampaign : campagne active (stale 1min, invalide post mutation)
 * - useCampaignMutations : start/pause/resume/close/abandon
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  campaignApi,
  playerProfileApi,
  type ImprovementCampaign,
  type PlayerProfile,
  type StartCampaignBody,
} from '@/lib/playerProfile'

// ─── Cache keys ────────────────────────────────────────────────────────────

export const profileKeys = {
  profile: (playerSlug: string, windowDays: number) =>
    ['playerProfile', playerSlug, windowDays] as const,
  activeCampaign: (playerSlug: string) =>
    ['playerProfile', 'campaign', 'active', playerSlug] as const,
  campaign: (playerSlug: string, id: string) =>
    ['playerProfile', 'campaign', playerSlug, id] as const,
}

// ─── Queries ───────────────────────────────────────────────────────────────

export function usePlayerProfile(playerSlug: string | undefined, windowDays = 30) {
  return useQuery<PlayerProfile>({
    queryKey: profileKeys.profile(playerSlug ?? '', windowDays),
    queryFn: () => playerProfileApi.getProfile(playerSlug!, windowDays),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

export function useActiveCampaign(playerSlug: string | undefined) {
  return useQuery<ImprovementCampaign | null>({
    queryKey: profileKeys.activeCampaign(playerSlug ?? ''),
    queryFn: () => campaignApi.getActive(playerSlug!),
    enabled: !!playerSlug,
    staleTime: 60 * 1000,
    retry: false,
  })
}

// ─── Mutations ─────────────────────────────────────────────────────────────

export function useCampaignMutations(playerSlug: string | undefined) {
  const qc = useQueryClient()
  const slug = playerSlug ?? ''
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: profileKeys.activeCampaign(slug) })
    qc.invalidateQueries({ queryKey: ['playerProfile', 'campaign', slug] })
  }
  return {
    start: useMutation({
      mutationFn: (body: StartCampaignBody) => campaignApi.start(slug, body),
      onSuccess: invalidate,
    }),
    pause: useMutation({
      mutationFn: (id: string) => campaignApi.pause(slug, id),
      onSuccess: invalidate,
    }),
    resume: useMutation({
      mutationFn: (id: string) => campaignApi.resume(slug, id),
      onSuccess: invalidate,
    }),
    close: useMutation({
      mutationFn: (id: string) => campaignApi.close(slug, id),
      onSuccess: invalidate,
    }),
    abandon: useMutation({
      mutationFn: (id: string) => campaignApi.abandon(slug, id),
      onSuccess: invalidate,
    }),
  }
}
