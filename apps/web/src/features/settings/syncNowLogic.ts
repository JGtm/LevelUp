/**
 * Logique pure de l'encadré « Synchronisation » : cadence de relecture de l'état du cycle et de
 * l'historique, texte du refus d'un lancement, lignes « dernière passe » et « prochaine passe ».
 * Aucune dépendance React.
 */
import { apiErrorCode } from '@/lib/api/client'
import { formatDate, intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { fillTemplate } from '@/lib/i18n/template'
import type { HistoryPass, HistorySyncStatus } from './syncNowQueries'
import type { SyncNowText } from './syncNowText'

/** Relecture de l'état du cycle, en ms : toutes les 3 s tant qu'un cycle tourne, jamais au repos. */
export const SYNC_CYCLE_POLL_MS = 3_000

/** Relecture de l'état de l'historique, en ms : toutes les 5 s pendant un cycle, toutes les 30 s pendant un rattrapage. */
export const HISTORY_POLL_RUNNING_MS = 5_000
export const HISTORY_POLL_CATCHING_UP_MS = 30_000

export function syncCycleRefetchMs(state: { running: boolean } | undefined): number | false {
  return state?.running ? SYNC_CYCLE_POLL_MS : false
}

/**
 * Cadence de relecture de l'état de l'historique : rapide tant qu'un cycle tourne, lente tant qu'au moins un
 * titre rattrape son historique, aucune au repos (une seule lecture).
 */
export function historyRefetchMs(status: HistorySyncStatus | undefined, cycleRunning: boolean): number | false {
  if (cycleRunning) return HISTORY_POLL_RUNNING_MS
  const catchingUp = status?.titles?.some((title) => title.state === 'catching_up') ?? false
  return catchingUp ? HISTORY_POLL_CATCHING_UP_MS : false
}

/** Le texte du refus d'un lancement, écrit ici dans chaque langue (jamais le message du serveur). */
export function startFailureText(t: SyncNowText, err: unknown): string {
  return apiErrorCode(err) === 'sync_cycle_busy' ? t.toastBusy : t.toastFailed
}

const DATE_TIME = { dateStyle: 'medium', timeStyle: 'short' } as const

/** Date et heure d'un instant, dans la langue de l'interface. */
export function formatInstant(iso: string | undefined, locale: Locale): string {
  return formatDate(iso, intlLocale(locale), DATE_TIME)
}

/** « Réussie, le 22 nov. 2026, 14:00 » ; le libellé d'issue arrive résolu du serveur. */
export function lastPassLine(t: SyncNowText, locale: Locale, pass: HistoryPass): string {
  return fillTemplate(t.lastPassValue, { status: pass.status_label, date: formatInstant(pass.at, locale) })
}

/** « Matchs ajoutés : 200 » ; seulement pour une passe qui a tourné (réussie ou partielle). */
export function lastPassAddedLine(t: SyncNowText, pass: HistoryPass): string | null {
  if (pass.status !== 'ok' && pass.status !== 'partial') return null
  return fillTemplate(t.lastPassAdded, { count: pass.matches_added })
}

/** La prochaine passe : cycle en cours, synchronisation automatique désactivée, heure prévue, ou pas encore planifiée. */
export function nextPassText(t: SyncNowText, locale: Locale, status: HistorySyncStatus, cycleRunning: boolean): string {
  if (cycleRunning) return t.nextPassRunning
  if (!status.auto_sync_enabled) return t.nextPassManualOnly
  if (!status.next_cycle_at) return t.nextPassUnknown
  return fillTemplate(t.nextPassAt, { date: formatInstant(status.next_cycle_at, locale) })
}
