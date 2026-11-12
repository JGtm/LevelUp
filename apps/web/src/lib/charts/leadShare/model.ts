/**
 * model.ts — LA MAINMISE VUE PAR L'AFFICHAGE : une part 0..1 (ADR 0006) ou rien.
 *
 * `null` / absent veut dire NON MESURÉ, jamais zéro : un mode à peu de points, un score non
 * couvert ou un film non lu n'ont pas de part. L'affichage dit « — », pas « 0 % ».
 */
import { intlLocale } from '@/lib/formatters/intlLocale'
import { formatPercent } from '@/lib/formatters/percent'
import type { Locale } from '@/lib/i18n/locale'
import { fillTemplate } from '@/lib/i18n/template'
import type { LeadShareText } from './text'

/** Une valeur servie par l'API : une fraction 0..1, ou absente (non mesuré). */
export type LeadShareValue = number | null | undefined

/** Le pourcentage ENTIER d'une part, ou le tiret quand elle n'est pas mesurée. */
export function formatLeadShare(share: LeadShareValue): string {
  return formatPercent(share, 0)
}

/**
 * « 3 retournements », au bon nombre : la catégorie de pluriel est celle de la LANGUE
 * (`Intl.PluralRules`), jamais un test sur 1 — en français zéro est au singulier, en anglais au
 * pluriel. Absent : chaîne vide (rien à dire).
 */
export function formatLeadChanges(
  changes: number | null | undefined,
  text: LeadShareText,
  locale: Locale,
): string {
  if (changes == null) return ''
  const one = new Intl.PluralRules(intlLocale(locale)).select(changes) === 'one'
  return fillTemplate(one ? text.changesOne : text.changesOther, { n: changes })
}

/** Une carte, un mois ou un match d'un graphe de Mainmise. */
export interface LeadSharePoint {
  key: string
  /** Libellé de l'axe. */
  label: string
  /** La part 0..1, ou null = non mesuré (colonne en pointillé, jamais zéro). */
  share: number | null
  /** Retournements du match, quand ils sont connus. */
  changes?: number | null
  /** Lignes ajoutées à l'infobulle (carte, mode, matchs mesurés...). */
  details?: string[]
}
