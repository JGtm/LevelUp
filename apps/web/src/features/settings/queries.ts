/**
 * Queries TanStack Query — Settings (Sprint 51 : split depuis setup/queries.ts).
 *
 * Contient uniquement les hooks liés à la configuration applicative.
 * Ne pas importer depuis features/setup/.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  AsyncJobStatus,
  BackfillStartRequest,
  BackupRunResult,
  BackupStatusResponse,
  SettingsResponse,
  UpdateSettingsRequest,
} from '@/lib/api/types'

export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: () => api.get<SettingsResponse>('/settings'),
    staleTime: 5 * 60 * 1000,
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  const setLocale = useAppShellStore((s) => s.setLocale)
  const currentLocale = useAppShellStore((s) => s.locale)
  return useMutation({
    mutationFn: (req: UpdateSettingsRequest) =>
      api.patch<SettingsResponse>('/settings', req),
    onSuccess: (data, variables) => {
      qc.setQueryData(queryKeys.settings, data)

      // Si l'utilisateur a changé `lang` dans Settings, propager au store
      // appShell (qui pousse le header X-LevelUp-Locale via setApiLocale) et
      // invalider les queries dont la réponse dépend de la locale (map_ui,
      // mode_ui, playlist_ui, labels narratifs…). Sans cela, le cache 5min
      // de useHomePage masquerait le changement.
      const newLang = (variables as Partial<SettingsResponse>).lang
      if (typeof newLang === 'string') {
        const next = newLang.toLowerCase().startsWith('en') ? 'en' : 'fr'
        if (next !== currentLocale) {
          setLocale(next)
          void qc.invalidateQueries() // toutes les queries — la locale touche
          // potentiellement chaque page. Côté coût : refetch on-demand,
          // pas un sync exhaustif, donc acceptable.
        }
      }
    },
  })
}

export function useScanMedia() {
  return useMutation({
    mutationFn: () => api.post<unknown>('/settings/media/scan', {}),
  })
}

export function useRecalculateSessions() {
  return useMutation({
    mutationFn: () => api.post<{ job_id: string }>('/settings/sessions/recalculate', {}),
  })
}

export function useStartBackfill() {
  return useMutation({
    mutationFn: (req: BackfillStartRequest) =>
      api.post<AsyncJobStatus>('/backfill/start', req),
  })
}

export function useBackupStatus() {
  return useQuery({
    queryKey: [...queryKeys.settings, 'backup-status'],
    queryFn: () => api.get<BackupStatusResponse>('/settings/backup/status'),
    staleTime: 30 * 1000,
  })
}

export function useRunBackup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<BackupRunResult>('/settings/backup/run', {}),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...queryKeys.settings, 'backup-status'] })
    },
  })
}
