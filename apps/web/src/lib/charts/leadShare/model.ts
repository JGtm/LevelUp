/**
 * model.ts — LA MAINMISE VUE PAR L'AFFICHAGE : une part 0..1 (ADR 0006) ou rien.
 *
 * `null` / absent veut dire NON MESURÉ, jamais zéro : un mode à peu de points, un score non
 * couvert ou un film non lu n'ont pas de part. L'affichage dit « — », pas « 0 % ».
 */
import { formatPercent } from '@/lib/formatters/percent'
import type { LeadShareText } from './text'

/** Une valeur servie par l'API : une fraction 0..1, ou absente (non mesuré). */
export type LeadShareValue = number | null | undefined

/** Le pourcentage ENTIER d'une part, ou le tiret quand elle n'est pas mesurée. */
export function formatLeadShare(share: LeadShareValue): string {
  return formatPercent(share, 0)
}

/** « 3 retournements », au bon nombre. Absent : chaîne vide (rien à dire). */
export function formatLeadChanges(changes: number | null | undefined, text: LeadShareText): string {
  if (changes == null) return ''
  return formatTemplate(changes === 1 ? text.changesOne : text.changesOther, { n: changes })
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

/** Moyenne des parts MESURÉES d'une liste de points, et leur nombre. `avg` null = aucune. */
export function averageOfMeasured(points: readonly LeadSharePoint[]): { avg: number | null; n: number } {
  const shares = points.flatMap((p) => (p.share == null ? [] : [p.share]))
  if (shares.length === 0) return { avg: null, n: 0 }
  return { avg: shares.reduce((a, b) => a + b, 0) / shares.length, n: shares.length }
}

/** Substitue `{nom}` dans un gabarit (les textes de ce module n'ont que des variables simples). */
export function formatTemplate(template: string, vars: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (whole, name: string) => (name in vars ? String(vars[name]) : whole))
}
