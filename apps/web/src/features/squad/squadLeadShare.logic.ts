/**
 * squadLeadShare.logic — LA MAINMISE DE LA SECTION DYNAMIQUE : un point par match du périmètre,
 * du plus ancien au plus récent (l'ordre est celui du serveur). L'escouade joue du même côté : la
 * part est celle de l'équipe du joueur principal.
 *
 * Un match sans `lead_share` est NON MESURÉ : la courbe se coupe, jamais un zéro.
 */
import type { SquadLeadSharePoint } from '@/lib/api/types'
import type { LeadSharePoint } from '@/lib/charts/leadShare/model'

export function squadLeadSharePoints(series: readonly SquadLeadSharePoint[] | null | undefined): LeadSharePoint[] {
  return (series ?? []).map((p, i) => ({
    key: p.match_id,
    label: `#${i + 1}${p.map_name ? `\n${p.map_name}` : ''}`,
    share: p.lead_share ?? null,
    changes: p.lead_changes ?? null,
    details: p.session_label ? [p.session_label] : [],
  }))
}
