/**
 * Lecture TanStack Query — emplacement, occupation et espace de l'archive des films
 * (GET /settings/film-archive). L'occupation est mesurée une fois puis mise en cache côté serveur
 * (10 min) : inutile de la relire plus souvent.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { components } from '@/lib/api/generated'

export type FilmArchiveStatus = components['schemas']['FilmArchiveStatus']

export function useFilmArchiveStatus() {
  return useQuery({
    queryKey: queryKeys.filmArchive,
    queryFn: () => api.get<FilmArchiveStatus>('/settings/film-archive'),
    staleTime: 60 * 1000,
  })
}
