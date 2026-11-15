/**
 * sessionLeadShare.logic — LA MAINMISE D'UNE SESSION : un point par match (colonne), et la ligne
 * de résumé (moyenne des matchs MESURÉS et leur nombre).
 *
 * Un match sans `lead_share` est NON MESURÉ (mode à peu de points, score non couvert, film non
 * lu) : sa colonne reste en pointillé, il n'entre ni au numérateur ni au dénominateur de la
 * moyenne — jamais un zéro. La moyenne et son nombre viennent du serveur (`entry.lead_share`,
 * `entry.lead_share_matches`) : ils ne se recalculent pas ici.
 */
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'
import { formatLeadShare, type LeadSharePoint } from '@/lib/charts/leadShare/model'
import type { LeadShareText } from '@/lib/charts/leadShare/text'
import { fillTemplate } from '@/lib/i18n/template'

import { sessionMatchAxisLabel } from './_shared'

/** Les colonnes du graphe, dans l'ordre des matchs de la session. */
export function sessionLeadSharePoints(matches: readonly SessionDetailMatchRow[]): LeadSharePoint[] {
  return matches.map((m, i) => ({
    key: m.match_id,
    label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
    share: m.lead_share ?? null,
    changes: m.lead_changes ?? null,
    details: m.mode_ui ? [m.mode_ui] : [],
  }))
}

/** « Moyenne 54 % sur 5 matchs », ou `null` quand aucun match n'est mesuré. */
export function sessionLeadShareSummary(
  entry: Pick<SessionCompareEntry, 'lead_share' | 'lead_share_matches'> | null | undefined,
  text: LeadShareText,
): string | null {
  if (entry?.lead_share == null || !entry.lead_share_matches) return null
  const n = entry.lead_share_matches
  return fillTemplate(n === 1 ? text.averageOne : text.averageOther, {
    pct: formatLeadShare(entry.lead_share),
    n,
  })
}
