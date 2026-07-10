/**
 * Queries ex-Lab encore en service (A3.5, DC-9) : seul le diagnostic
 * d'instance reste consommé (panneau DiagnosticsPanel dans l'onglet admin
 * Données). Les explorateurs Resources / Waypoint sont retirés avec leurs
 * endpoints back (garde-rail : lab-removal.guard.test.ts).
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { LabDiagnosticsResponse } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

export function useLabDiagnostics(enabled = true) {
  return useQuery({
    queryKey: queryKeys.labDiagnostics,
    queryFn: () => api.get<LabDiagnosticsResponse>('/lab/diagnostics'),
    enabled,
    staleTime: 60 * 1000,
  })
}
