/**
 * Queries TanStack Query — Settings (Sprint 51 : split depuis setup/queries.ts).
 *
 * Contient uniquement les hooks liés à la configuration applicative.
 * Ne pas importer depuis features/setup/.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { parseRouteSegments } from '@/lib/title-routing'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  AsyncJobStatus,
  BackfillStartRequest,
  BackupRunResult,
  BackupStatusResponse,
  PlayersListResponse,
  SettingsResponse,
  UpdateSettingsRequest,
} from '@/lib/api/types'

export function useSettings() {
  // En démo, la langue est un choix purement client-side (le PATCH /settings est
  // refusé et GET /settings renvoie la valeur figée et partagée). On superpose
  // donc la locale du store appShell sur la réponse : sans cela, le refetch
  // /settings déclenché après un switch écraserait le choix du visiteur dans le
  // sélecteur de langue. Hors démo, la réponse serveur fait autorité (select nop).
  const demoMode = useAppShellStore((s) => s.demoMode)
  const demoLocale = useAppShellStore((s) => s.locale)
  return useQuery({
    queryKey: queryKeys.settings,
    queryFn: () => api.get<SettingsResponse>('/settings'),
    staleTime: 5 * 60 * 1000,
    select: demoMode
      ? (data: SettingsResponse): SettingsResponse => ({ ...data, lang: demoLocale })
      : undefined,
  })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  const setLocale = useAppShellStore((s) => s.setLocale)
  const currentLocale = useAppShellStore((s) => s.locale)
  const demoMode = useAppShellStore((s) => s.demoMode)
  const navigate = useNavigate()
  return useMutation({
    mutationFn: async (req: UpdateSettingsRequest) => {
      // En démo, le PATCH /settings est refusé (422 — settings figés et partagés
      // entre visiteurs). Le seul réglage modifiable est la langue : on l'applique
      // client-side (onSuccess ci-dessous), sans toucher au serveur. Les autres
      // champs sont ignorés (no-op) — l'UI les grise de toute façon.
      if (demoMode) {
        const current =
          qc.getQueryData<SettingsResponse>(queryKeys.settings) ?? ({} as SettingsResponse)
        const lang = (req as Partial<SettingsResponse>).lang
        return typeof lang === 'string' ? { ...current, lang } : current
      }
      return api.patch<SettingsResponse>('/settings', req)
    },
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

          // Réalignement du segment lang (D-12, I10) : si l'URL courante est
          // TITLE-SCOPED (seg.titleSlug), réécrire — ou INJECTER si absent — le segment
          // lang vers la nouvelle locale (navigate REPLACE, même route). La condition
          // couvre les DEUX cas : segment présent mais périmé, ET segment absent sur une
          // URL title-scoped. Sans ce réalignement, le segment (source de vérité) et la
          // réconciliation du layout titre (5a) re-forceraient aussitôt l'ancienne locale.
          // La page Settings étant AGNOSTIQUE (hors segment, D-3), l'URL y est '/settings'
          // → seg.titleSlug undefined → NO-OP : ce chemin ne s'active que depuis une
          // surface title-scoped offrant un sélecteur de langue.
          const seg = parseRouteSegments(window.location.pathname)
          if (seg.titleSlug && seg.lang !== next) {
            void navigate({ to: '.', params: (prev) => ({ ...prev, lang: next }), replace: true })
          }
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

// ─── Sélection par titre (onglet « Jeux ») ───────────────────────────────────

export interface PlayerTitleStatus {
  slug: string
  name: string
  status: 'active' | 'coming_soon' | 'archived'
  enrolled: boolean
  syncEnabled: boolean
}

interface TitleSyncResult {
  gamertag: string
  title_slug: string
  sync_enabled: boolean
}

interface TitlePurgeResult {
  gamertag: string
  title_slug: string
  data_removed: boolean
}

/**
 * usePlayerTitles agrège, pour le joueur courant, le statut sync par titre.
 * Source : GET /players par titre (header X-LevelUp-Title override). GET /players
 * filtre par titre courant, donc on itère les titres disponibles (Option B, N
 * petit). Chaque appel est résilient : un titre en erreur → enrolled=false.
 */
export function usePlayerTitles() {
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  return useQuery({
    queryKey: queryKeys.playerTitles(playerSlug),
    enabled: playerSlug !== '',
    queryFn: async (): Promise<PlayerTitleStatus[]> => {
      const titles = availableTitles.filter((t) => t.status !== 'archived')
      return Promise.all(
        titles.map(async (t): Promise<PlayerTitleStatus> => {
          try {
            const resp = await api.get<PlayersListResponse>('/players', { 'X-LevelUp-Title': t.slug })
            const entry = (resp.items ?? []).find((p) => p.player_slug === playerSlug)
            return {
              slug: t.slug,
              name: t.name,
              status: t.status,
              enrolled: entry != null,
              syncEnabled: entry?.sync_enabled ?? false,
            }
          } catch {
            return { slug: t.slug, name: t.name, status: t.status, enrolled: false, syncEnabled: false }
          }
        }),
      )
    },
  })
}

/** useSetTitleSync bascule actif/pause d'un titre du joueur courant. */
export function useSetTitleSync() {
  const qc = useQueryClient()
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  return useMutation({
    mutationFn: ({ slug, enabled }: { slug: string; enabled: boolean }) =>
      api.patch<TitleSyncResult>(
        `/profiles/${encodeURIComponent(playerSlug)}/titles/${encodeURIComponent(slug)}/sync`,
        { enabled },
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.playerTitles(playerSlug) })
      void qc.invalidateQueries({ queryKey: queryKeys.bootstrap })
    },
  })
}

/** usePurgeTitleData supprime le profil + les données d'un titre du joueur courant. */
export function usePurgeTitleData() {
  const qc = useQueryClient()
  const playerSlug = useAppShellStore((s) => s.currentPlayer?.player_slug ?? '')
  return useMutation({
    mutationFn: ({ slug }: { slug: string }) =>
      api.delete<TitlePurgeResult>(
        `/profiles/${encodeURIComponent(playerSlug)}/titles/${encodeURIComponent(slug)}/data`,
      ),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.playerTitles(playerSlug) })
      void qc.invalidateQueries({ queryKey: queryKeys.bootstrap })
    },
  })
}
