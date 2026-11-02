/**
 * Lecture TanStack Query — état vivant de l'intégration au poste de l'application de bureau
 * (GET /settings/desktop) : le jeu tourne-t-il. Relue toutes les 15 s, la période de sondage du serveur,
 * tant que la carte est affichée.
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { components } from '@/lib/api/generated'

export type DesktopStatus = components['schemas']['DesktopStatus']

/** Période de relecture de l'état vivant : celle du sondage des processus côté serveur. */
const DESKTOP_STATUS_REFETCH_MS = 15_000

export function useDesktopStatus(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.desktopStatus,
    queryFn: () => api.get<DesktopStatus>('/settings/desktop'),
    staleTime: DESKTOP_STATUS_REFETCH_MS,
    refetchInterval: DESKTOP_STATUS_REFETCH_MS,
    enabled,
  })
}
