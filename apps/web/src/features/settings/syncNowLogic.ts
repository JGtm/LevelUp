/**
 * Logique pure de l'encadré « Synchronisation » : cadence de relecture de l'état du cycle, texte du refus
 * d'un lancement. Aucune dépendance React.
 */
import { apiErrorCode } from '@/lib/api/client'
import type { SyncNowText } from './syncNowText'

/** Relecture de l'état du cycle, en ms : toutes les 3 s tant qu'un cycle tourne, jamais au repos. */
export const SYNC_CYCLE_POLL_MS = 3_000

export function syncCycleRefetchMs(state: { running: boolean } | undefined): number | false {
  return state?.running ? SYNC_CYCLE_POLL_MS : false
}

/** Le texte du refus d'un lancement, écrit ici dans chaque langue (jamais le message du serveur). */
export function startFailureText(t: SyncNowText, err: unknown): string {
  return apiErrorCode(err) === 'sync_cycle_busy' ? t.toastBusy : t.toastFailed
}
