/**
 * Mutations des actions de résolution qualité données.
 * Invalidation centralisée via invalidateDataQuality (la ligne résolue
 * disparaît des listes, les compteurs se mettent à jour).
 */
import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  AssetTranslationRequest,
  CatalogRefreshResult,
  RegistryNamesBackfillResult,
  ResolveResult,
} from '@/lib/api/types'

/** Invalide compteurs + toutes les listes d'inconnus (+ l'overview). */
export function invalidateDataQuality(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: queryKeys.adminDataQuality })
  void queryClient.invalidateQueries({ queryKey: ['admin', 'data-quality', 'issues'] })
  void queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringOverview })
}

export function useRunRegistryNamesBackfill() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (dryRun: boolean) =>
      api.post<RegistryNamesBackfillResult>('/admin/actions/registry-names/backfill', {
        dry_run: dryRun,
      }),
    onSuccess: (_res, dryRun) => {
      if (!dryRun) invalidateDataQuality(queryClient)
    },
  })
}

export function useResolveModeTranslation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: { mode_en: string; name_fr: string }) =>
      api.post<ResolveResult>('/admin/actions/translations/mode', req),
    onSuccess: () => invalidateDataQuality(queryClient),
  })
}

export function useResolveAssetTranslation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: AssetTranslationRequest) =>
      api.post<ResolveResult>('/admin/actions/translations/asset', req),
    onSuccess: () => invalidateDataQuality(queryClient),
  })
}

export function useRunPlayerConvergence() {
  return useMutation({
    mutationFn: (playerSlug: string) =>
      api.post('/admin/actions/convergence/run', { player_slug: playerSlug }),
  })
}

export function useRunCatalogRefresh() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<CatalogRefreshResult>('/admin/actions/catalog/refresh', {}),
    onSuccess: () => invalidateDataQuality(queryClient),
  })
}
