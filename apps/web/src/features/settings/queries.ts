/**
 * Queries TanStack Query — Settings (Sprint 51 : split depuis setup/queries.ts).
 *
 * Contient uniquement les hooks liés à la configuration applicative.
 * Ne pas importer depuis features/setup/.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { SettingsResponse, UpdateSettingsRequest } from '@/lib/api/types'

export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: () => api.get<SettingsResponse>('/settings'),
    staleTime: 5 * 60 * 1000,
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: UpdateSettingsRequest) =>
      api.patch<SettingsResponse>('/settings', req),
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.settings, data)
    },
  })
}
