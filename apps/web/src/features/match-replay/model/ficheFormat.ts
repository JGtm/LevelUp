/**
 * ficheFormat.ts — L'ÉCRITURE DES NOMBRES DE LA FICHE : un entier, un pour-cent entier, une durée, dans la langue
 * de la page. Fichier dédié : un fichier de composant n'exporte que des composants.
 */
import { formatNumber, formatSignedFixed } from '@/lib/formatters/number'
import { formatClock } from '@/lib/replay/replayLogic'

import type { ReplayLocale } from '../i18n/i18n'

/** Un entier groupé selon la langue (« 1 480 »). */
export function formatCount(n: number, locale: ReplayLocale): string {
  return formatNumber(Math.round(n), locale, 0)
}

/** Un entier AVEC son signe (« +2 », « −1 », « ±0 »), pour un écart. */
export function formatDifferential(n: number): string {
  return formatSignedFixed(n, 0)
}

/** Un pour-cent entier, insécable : « 41 % ». `pct` est entre 0 et 100. */
export function formatPercent(pct: number, locale: ReplayLocale): string {
  return `${formatCount(pct, locale)}\u00a0%`
}

/** Une durée en millisecondes, à la seconde : « 0:27 ». */
export function formatDuration(ms: number): string {
  return formatClock(ms)
}
