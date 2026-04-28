import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AssetMeta } from '@/lib/api/types'

const STALE_MS = 5 * 60 * 1000

export function useAssetMaps(titleSlug: string, search: string) {
  return useQuery({
    queryKey: queryKeys.assetMaps(titleSlug, search),
    queryFn: () => {
      const params = search ? `?q=${encodeURIComponent(search)}` : ''
      return api.get<AssetMeta[]>(`/assets/${titleSlug}/maps${params}`)
    },
    enabled: !!titleSlug,
    staleTime: STALE_MS,
  })
}

export function useAssetWeapons(titleSlug: string, search: string) {
  return useQuery({
    queryKey: queryKeys.assetWeapons(titleSlug, search),
    queryFn: () => {
      const params = search ? `?q=${encodeURIComponent(search)}` : ''
      return api.get<AssetMeta[]>(`/assets/${titleSlug}/weapons${params}`)
    },
    enabled: !!titleSlug,
    staleTime: STALE_MS,
  })
}
