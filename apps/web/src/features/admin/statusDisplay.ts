/**
 * statusDisplay.ts — mapping pur statut système → token sémantique + clé i18n
 * (StatusBadge) + libellés des types de jobs. Testable sans React.
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import type { JobStatus, SchedulerOutcome } from '@/lib/api/types'
import type { AdminLocale } from './format'

/** Statuts affichables par StatusBadge (jobs + outcomes scheduler + génériques). */
export type AdminStatus =
  | 'ok'
  | 'warning'
  | 'error'
  | 'running'
  | 'queued'
  | 'succeeded'
  | 'failed'
  | 'cancelled'
  | 'interrupted'
  | 'idle'
  | 'skipped'

export interface StatusDisplay {
  /** Token de couleur (undefined → neutre muted). */
  token?: SemanticToken
  labelKey: AdminManifestKey
  /** Dot carré animé (états actifs). */
  pulse: boolean
}

const STATUS_DISPLAY: Record<AdminStatus, StatusDisplay> = {
  ok: { token: 'success', labelKey: 'admin.status.ok', pulse: false },
  warning: { token: 'warning', labelKey: 'admin.status.warning', pulse: false },
  error: { token: 'destructive', labelKey: 'admin.status.error', pulse: false },
  running: { token: 'info', labelKey: 'admin.status.running', pulse: true },
  queued: { token: 'info', labelKey: 'admin.status.queued', pulse: true },
  succeeded: { token: 'success', labelKey: 'admin.status.succeeded', pulse: false },
  failed: { token: 'destructive', labelKey: 'admin.status.failed', pulse: false },
  cancelled: { labelKey: 'admin.status.cancelled', pulse: false },
  interrupted: { token: 'warning', labelKey: 'admin.status.interrupted', pulse: false },
  idle: { labelKey: 'admin.status.idle', pulse: false },
  skipped: { labelKey: 'admin.status.skipped', pulse: false },
}

/** Affichage d'un AdminStatus — valeur inconnue → neutre 'idle' (jamais de crash). */
export function statusDisplay(status: string): StatusDisplay {
  return STATUS_DISPLAY[(status as AdminStatus) in STATUS_DISPLAY ? (status as AdminStatus) : 'idle']
}

/** Mappe un JobStatus backend vers un AdminStatus de badge. */
export function jobStatusToAdminStatus(status: JobStatus | string): AdminStatus {
  switch (status) {
    case 'queued':
      return 'queued'
    case 'running':
      return 'running'
    case 'succeeded':
      return 'succeeded'
    case 'failed':
      return 'failed'
    case 'cancelled':
      return 'cancelled'
    case 'interrupted':
      return 'interrupted'
    default:
      return 'idle'
  }
}

/**
 * Seuil de fraîcheur du flux de présence : au-delà, le daemon tourne mais ne
 * reçoit plus d'event. Le REST poll tique toutes les 10 s (cf.
 * rest_poller.restPollInterval) ; 90 s = 9 polls ratés d'affilée, marge
 * suffisante pour absorber un backoff transitoire (rate-limit/réseau) sans
 * crier au loup, assez court pour repérer un flux franchement figé.
 */
export const WATCHER_STALE_MS = 90_000

/**
 * watcherLivenessStatus — vivacité du flux de présence, indépendante de l'état
 * FSM des joueurs : un daemon "running" qui ne reçoit plus d'event est mort en
 * pratique (polls en échec en boucle). `nowMs` est injecté pour la testabilité.
 *
 *  - daemon arrêté            → 'idle'   (non pertinent)
 *  - aucun event depuis boot  → 'warning' (en attente du premier snapshot)
 *  - dernier event récent     → 'ok'
 *  - dernier event trop vieux → 'error'  (flux figé)
 */
export function watcherLivenessStatus(
  lastEventAt: string | undefined,
  daemonRunning: boolean,
  nowMs: number = Date.now(),
): AdminStatus {
  if (!daemonRunning) return 'idle'
  if (!lastEventAt) return 'warning'
  const ts = new Date(lastEventAt).getTime()
  if (Number.isNaN(ts)) return 'warning'
  return nowMs - ts <= WATCHER_STALE_MS ? 'ok' : 'error'
}

/** Mappe un outcome scheduler ("ok"/"skipped"/"failed") vers un AdminStatus. */
export function schedulerOutcomeToAdminStatus(outcome: SchedulerOutcome | string): AdminStatus {
  switch (outcome) {
    case 'ok':
      return 'ok'
    case 'skipped':
      return 'skipped'
    case 'failed':
      return 'failed'
    default:
      return 'idle'
  }
}

/**
 * Libellé FR/EN des types de jobs connus (JobType backend). Type inconnu →
 * type brut (mono côté UI) : repérable sans crash quand un nouveau type
 * apparaît côté Go.
 */
const JOB_TYPE_LABELS: Record<string, { fr: string; en: string }> = {
  setup_smoke_test: { fr: 'Test de configuration', en: 'Setup smoke test' },
  initial_sync: { fr: 'Sync initiale', en: 'Initial sync' },
  delta_sync_all: { fr: 'Sync delta (tous)', en: 'Delta sync (all)' },
  backfill: { fr: 'Backfill', en: 'Backfill' },
  reindex_media: { fr: 'Réindexation médias', en: 'Media reindex' },
  scan_media: { fr: 'Scan médias', en: 'Media scan' },
  transcode_media: { fr: 'Transcodage média', en: 'Media transcode' },
  sessions_recalculate: { fr: 'Recalcul des sessions', en: 'Sessions recalculation' },
  openspartan_import: { fr: 'Import OpenSpartan', en: 'OpenSpartan import' },
  forced_sync_cycle: { fr: 'Cycle de sync forcé', en: 'Forced sync cycle' },
  player_convergence: { fr: 'Convergence joueur', en: 'Player convergence' },
  catalog_refresh: { fr: 'Refresh catalogue', en: 'Catalog refresh' },
  replay_build: { fr: 'Construction du rejeu 2D', en: '2D replay build' },
}

export function jobTypeLabel(jobType: string, locale: AdminLocale): string {
  return JOB_TYPE_LABELS[jobType]?.[locale] ?? jobType
}
