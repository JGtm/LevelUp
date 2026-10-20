import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AssetMeta } from '@/lib/api/types'

const STALE_MS = 5 * 60 * 1000

// `isOpen` : le tiroir est monté par l'AppShell sur TOUTES les pages ; ses trois
// catalogues ne partent que tiroir ouvert (lot perf L4a, D4.5, 2026-09-23), puis se
// comportent comme avant (les trois onglets chargés, la recherche sur l'onglet actif).

export function useAssetMaps(titleSlug: string, search: string, isOpen: boolean) {
  return useQuery({
    queryKey: queryKeys.assetMaps(titleSlug, search),
    queryFn: () => {
      const params = search ? `?q=${encodeURIComponent(search)}` : ''
      return api.get<AssetMeta[]>(`/assets/${titleSlug}/maps${params}`)
    },
    enabled: isOpen && !!titleSlug,
    staleTime: STALE_MS,
  })
}

export function useAssetWeapons(titleSlug: string, search: string, isOpen: boolean) {
  return useQuery({
    queryKey: queryKeys.assetWeapons(titleSlug, search),
    queryFn: () => {
      const params = search ? `?q=${encodeURIComponent(search)}` : ''
      return api.get<AssetMeta[]>(`/assets/${titleSlug}/weapons${params}`)
    },
    enabled: isOpen && !!titleSlug,
    staleTime: STALE_MS,
  })
}

export function useAssetMedals(titleSlug: string, search: string, isOpen: boolean) {
  return useQuery({
    queryKey: queryKeys.assetMedals(titleSlug, search),
    queryFn: () => {
      const params = search ? `?q=${encodeURIComponent(search)}` : ''
      return api.get<AssetMeta[]>(`/assets/${titleSlug}/medals${params}`)
    },
    enabled: isOpen && !!titleSlug,
    staleTime: STALE_MS,
  })
}
