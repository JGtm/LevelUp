/**
 * Lecture de l'état de synchronisation pour la carte « Synchronisation » : ce que la carte dit et ce
 * qu'elle propose, décidé en un point à partir de la réponse du serveur.
 */
import type { PlayerSyncStatus } from '@/lib/api/types'

/** Ce que la carte dit de l'historique ; `hidden` quand le serveur n'en sait rien de fiable (démo, état non lu). */
export type HistoryState = 'complete' | 'catching_up' | 'paused_incomplete' | 'hidden'

export interface SyncCardView {
  /** Une synchronisation du joueur est en vol (quelle qu'en soit la source). */
  running: boolean
  /** Minutes restantes avant qu'un lancement manuel soit de nouveau accepté ; 0 hors délai. */
  cooldownMinutes: number
  /** Le bouton peut lancer : droit d'écriture, rien en vol, aucun délai, carte non figée. */
  canStart: boolean
  /** La synchronisation de ce titre est en pause pour ce joueur : aucun bouton, la réactivation se fait ailleurs. */
  paused: boolean
  history: HistoryState
}

interface SyncCardInput {
  /** Réponse du serveur, déjà vérifiée comme étant celle du joueur affiché. */
  status: PlayerSyncStatus | undefined
  canWrite: boolean
  /** `sync_enabled` du profil affiché ; `undefined` = profil inconnu (pas de pause déclarée), même règle que l'accueil. */
  syncEnabled: boolean | undefined
  /** Démo : la carte est figée et le serveur sert un état d'historique qui ne dit rien du réel. */
  frozen: boolean
  /** Un lancement est parti et son effet n'est pas encore relu. */
  starting: boolean
}

export function resolveSyncCardView({ status, canWrite, syncEnabled, frozen, starting }: SyncCardInput): SyncCardView {
  const paused = syncEnabled === false
  const running = starting || status?.syncing === true
  const cooldownSec = status?.cooldown_retry_after_s ?? 0
  // Titre en pause : rien ne rattrape, la carte ne dit pas qu'un rattrapage est en cours.
  const history: HistoryState =
    frozen || !status ? 'hidden' : status.history_complete ? 'complete' : paused ? 'paused_incomplete' : 'catching_up'
  return {
    running,
    cooldownMinutes: cooldownSec > 0 ? Math.ceil(cooldownSec / 60) : 0,
    canStart: canWrite && !paused && !frozen && !running && cooldownSec <= 0,
    paused,
    history,
  }
}

const TERMINAL_JOB_STATUSES: ReadonlySet<string> = new Set(['succeeded', 'failed', 'cancelled', 'interrupted'])

/** Vrai quand `status` porte la fin du job `jobId` (le job du lancement fait depuis cette carte). */
export function jobEnded(status: PlayerSyncStatus | undefined, jobId: string | null): boolean {
  if (!status || !jobId || status.syncing) return false
  return status.job?.job_id === jobId && TERMINAL_JOB_STATUSES.has(status.job.status)
}
