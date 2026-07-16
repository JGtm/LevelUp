/**
 * prefetch.ts — prefetch spéculatif sur hover des liens de navigation L1.
 *
 * Quand l'utilisateur survole un onglet, on lance un prefetchQuery pour les
 * données de cette section. TanStack Query ne refetch pas si les données sont
 * déjà fraîches (staleTime respecté). Le coût est donc nul si le cache est chaud.
 */
import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { api } from '@/lib/api/client'
import { queryKeys } from './keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  CareerPageResponse,
  HomePageResponse,
} from '@/lib/api/types'

const PREFETCH_STALE_MS = 30 * 60 * 1000 // cohérent avec career/home staleTime

/**
 * Retourne un handler onMouseEnter à brancher sur chaque lien L1.
 * Seules les sections avec données lourdes sont prefetchées (home, career).
 */
export function useNavPrefetch(playerSlug: string) {
  const queryClient = useQueryClient()

  const prefetchHome = useCallback(() => {
    if (!playerSlug) return
    void queryClient.prefetchQuery({
      queryKey: queryKeys.home(playerSlug, useAppShellStore.getState().currentTitleSlug),
      queryFn: () => api.get<HomePageResponse>(`/players/${playerSlug}/pages/home`),
      staleTime: PREFETCH_STALE_MS,
    })
  }, [queryClient, playerSlug])

  const prefetchCareer = useCallback(() => {
    if (!playerSlug) return
    void queryClient.prefetchQuery({
      queryKey: queryKeys.career(playerSlug),
      queryFn: () => api.get<CareerPageResponse>(`/players/${playerSlug}/pages/career`),
      staleTime: PREFETCH_STALE_MS,
    })
  }, [queryClient, playerSlug])

  return useCallback(
    (sectionKey: string) => {
      switch (sectionKey) {
        case 'home':
          prefetchHome()
          break
        case 'career':
          prefetchCareer()
          break
        default:
          break
      }
    },
    [prefetchHome, prefetchCareer],
  )
}
