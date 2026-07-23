/**
 * actionJournal — journal des actions globales admin (dernière exécution /
 * issue / déclencheur par action, persisté hors DuckDB, survit au reboot — C2).
 *
 * Hook de lecture partagé (TanStack dédupe par query key → un seul GET pour
 * toutes les lignes) + constantes des noms d'action. Le composant d'affichage
 * vit dans ActionLastRun.tsx (fichier distinct : fast-refresh).
 */
import { useQuery, type QueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AdminActionJournalResponse } from '@/lib/api/types'

// Noms canoniques des actions journalisées — MIROIR des constantes Go
// (internal/platform/adminstate/action_journal.go). Toute divergence = ligne
// « Jamais exécutée » qui ne se remplit jamais.
export const ACTION_REGISTRY_NAMES = 'registry_names_backfill'
export const ACTION_CATALOG_REFRESH = 'catalog_refresh'
export const ACTION_LYING_BITS_RESET = 'lying_bits_reset'
export const ACTION_CATALOG_UGC_DRAIN = 'catalog_ugc_drain'
export const ACTION_DATA_HEALTH = 'data_health'
export const ACTION_SYNC_CYCLE = 'sync_cycle'

/** Lecture du journal des actions globales (polling tranquille aligné overview). */
export function useActionJournal() {
  return useQuery({
    queryKey: queryKeys.adminActionJournal,
    queryFn: () => api.get<AdminActionJournalResponse>('/admin/actions/journal'),
    refetchInterval: 30_000,
    staleTime: 15_000,
    retry: false,
  })
}

/** Invalide le journal après une action (rafraîchit « Dernière exécution »). */
export function invalidateActionJournal(qc: QueryClient) {
  void qc.invalidateQueries({ queryKey: queryKeys.adminActionJournal })
}
