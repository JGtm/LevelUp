/**
 * Hooks React Query — arcs Prestige.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { prestigeApi, type CreateArcBody } from '@/lib/prestige'

export const arcKeys = {
  list: (userId: string, titleSlug: string) =>
    ['prestige', 'arcs', userId, titleSlug] as const,
  one: (id: string) => ['prestige', 'arc', id] as const,
}

export function useArcs(userId: string, titleSlug: string) {
  return useQuery({
    queryKey: arcKeys.list(userId, titleSlug),
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
      qc.invalidateQueries({ queryKey: arcKeys.list(userId, titleSlug) })
    },
  })
}
