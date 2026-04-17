/**
 * Queries TanStack Query — Médias (Slice 8).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MediaQueryRequest, MediaPageResponse, MediaUploadResponse } from '@/lib/api/types'

export function useMediaPage(
  playerSlug: string,
  request: MediaQueryRequest,
  page: number,
) {
  return useQuery({
    queryKey: queryKeys.media(playerSlug, page),
    queryFn: () =>
      api.post<MediaPageResponse>(
        `/players/${playerSlug}/pages/media`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
  })
}

/** Mutation upload : envoie des fichiers et invalide toutes les pages médias. */
export function useUploadMedia(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (files: File[]) => {
      const form = new FormData()
      files.forEach((f) => form.append('files', f, f.name))
      return api.postForm<MediaUploadResponse>(
        `/players/${playerSlug}/media/upload`,
        form,
      )
    },
    onSuccess: () => {
      // Invalide toutes les pages media du joueur (préfixe ['media', slug])
      queryClient.invalidateQueries({
        queryKey: queryKeys.mediaBase(playerSlug),
      })
    },
  })
}
