/**
 * playerPanelFormat.ts — LE TEXTE D'UNE VALEUR DU PANNEAU DE JOUEUR : un entier, un pourcentage à une
 * décimale, ou la lacune. Fichier dédié : un fichier de composant n'exporte que des composants.
 *
 * UNE VALEUR NON LUE EST UN TIRET (`NO_VALUE`), jamais un zéro : la feuille ne porte pas toujours
 * chaque champ, et un 0 affirmerait une mesure qu'on n'a pas. Un zéro MESURÉ s'écrit « 0 ».
 */
import { formatNumber } from '@/lib/formatters/number'

import type { ReplayLocale } from '../i18n/i18n'
import type { PanelRow } from './playerPanelLogic'

/** La lacune d'une valeur que la feuille ne porte pas. */
export const NO_VALUE = '—'

export function formatPanelValue(row: Pick<PanelRow, 'format' | 'value'>, locale: ReplayLocale): string {
  if (row.value === null) return NO_VALUE
  return row.format === 'percent'
    ? `${formatNumber(row.value, locale, 1)}\u00a0%`
    : formatNumber(row.value, locale, 0)
}
