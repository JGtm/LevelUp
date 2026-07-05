/**
 * Hooks React Query — arcs Prestige.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { prestigeApi, type CreateArcBody } from '@/lib/prestige'
import { queryKeys } from '@/lib/query/keys'

export function useArcs(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: queryKeys.arc.list(userId, titleSlug),
    queryFn: () => prestigeApi.listArcs(userId, titleSlug),
    retry: false,
    enabled: !!userId && !!titleSlug,
  })
}

export function useCreateArc(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateArcBody) => prestigeApi.createArc(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.arc.list(userId, titleSlug) })
    },
  })
}

export function useDeleteArc(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    // cascade=true supprime aussi les objectifs ; false les détache.
    mutationFn: ({ id, cascade }: { id: string; cascade: boolean }) =>
      prestigeApi.deleteArc(id, userId, cascade),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.arc.list(userId, titleSlug) })
      // La cascade abandonne/supprime des objectifs → rafraîchir aussi les défis.
      qc.invalidateQueries({ queryKey: queryKeys.challenge.list(userId, titleSlug) })
    },
  })
}

export function useArcPresets(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: queryKeys.arc.presets(userId, titleSlug),
    queryFn: () => prestigeApi.listArcPresets(userId, titleSlug),
    retry: false,
    enabled: !!userId && !!titleSlug,
    staleTime: 5 * 60_000, // catalogue versionné : cache long
  })
}

export function useAdoptArcPreset(userId: string, titleSlug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (presetId: string) => prestigeApi.adoptArcPreset(presetId, userId, titleSlug),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.arc.list(userId, titleSlug) })
      // L'adoption crée des objectifs → rafraîchir aussi les défis.
      qc.invalidateQueries({ queryKey: queryKeys.challenge.list(userId, titleSlug) })
    },
  })
}
