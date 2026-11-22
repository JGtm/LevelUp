/**
 * Logique pure de l'encadré « Synchronisation » : cadence de relecture de l'état du cycle et de
 * l'historique, texte du refus d'un lancement, lignes « dernière passe » et « prochaine passe ».
 * Aucune dépendance React.
 */
import { analysisPendingLabelKey } from '@/components/ui/match-card-presentation'
import { apiErrorCode } from '@/lib/api/client'
import { formatDate, intlLocale } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'
import { fillTemplate } from '@/lib/i18n/template'
import type { HistoryPass, HistorySyncStatus, TitleHistoryStatus } from './syncNowQueries'
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
 * titre rattrape son historique ou a des films à analyser, aucune au repos (une seule lecture).
 */
export function historyRefetchMs(status: HistorySyncStatus | undefined, cycleRunning: boolean): number | false {
  if (cycleRunning) return HISTORY_POLL_RUNNING_MS
  const catchingUp = status?.titles?.some((title) => title.state === 'catching_up' || title.films_pending) ?? false
  return catchingUp ? HISTORY_POLL_CATCHING_UP_MS : false
}

/**
 * Libellé de l'état d'un titre. L'historique n'est dit complet que si aucun film n'attend son analyse : sinon
 * l'écran reprend l'indication des tuiles de match (« Analyse en cours » pendant un cycle, « Analyse en
 * attente » sinon), le même texte dans la même langue. Les autres états gardent le libellé du serveur.
 */
export function historyStateLabel(title: TitleHistoryStatus, locale: Locale, cycleRunning: boolean): string {
  if (title.state === 'complete' && title.films_pending) {
    return formatMessage(commonManifest, analysisPendingLabelKey('badge', cycleRunning), locale)
  }
  return title.state_label
}

/** Code du refus d'un lancement trop tôt après un cycle : l'écran n'en fait pas une erreur, il montre le délai. */
export const COOLDOWN_CODE = 'sync_cycle_cooldown'

/** Secondes restantes avant l'heure donnée (arrondies au-dessus), 0 si elle est passée ou absente. */
export function remainingSeconds(availableAt: string | undefined, nowMs: number): number {
  if (!availableAt) return 0
  const at = Date.parse(availableAt)
  if (Number.isNaN(at)) return 0
  return Math.max(0, Math.ceil((at - nowMs) / 1000))
}

/** « 1 min 40 » au-delà d'une minute, « 40 s » en dessous : mêmes unités dans toutes les langues. */
export function formatRemaining(seconds: number): string {
  if (seconds < 60) return `${seconds} s`
  return `${Math.floor(seconds / 60)} min ${String(seconds % 60).padStart(2, '0')}`
}

const HOUR_MINUTE = { hour: '2-digit', minute: '2-digit' } as const

/** Heure d'un instant (sans date), dans la langue de l'interface. */
function formatClock(iso: string | undefined, locale: Locale): string {
  return formatDate(iso, intlLocale(locale), HOUR_MINUTE)
}

/** Libellé du bouton : cycle en cours, délai restant, ou l'action. */
export function buttonLabel(t: SyncNowText, running: boolean, remaining: number): string {
  if (running) return t.buttonRunning
  if (remaining > 0) return fillTemplate(t.buttonCooldown, { remaining: formatRemaining(remaining) })
  return t.button
}

/** Ligne sous le bouton : cycle en cours, ou fin du dernier cycle et prochain lancement ; rien sinon. */
export function manualStatusLine(
  t: SyncNowText,
  locale: Locale,
  cycle: { last_cycle_at?: string; manual_available_at?: string } | undefined,
  running: boolean,
  remaining: number,
): string | null {
  if (running) return t.statusRunning
  if (remaining <= 0 || !cycle) return null
  return fillTemplate(t.statusCooldown, {
    last: formatClock(cycle.last_cycle_at, locale),
    next: formatClock(cycle.manual_available_at, locale),
  })
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
