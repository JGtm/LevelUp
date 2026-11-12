/**
 * matchLeadShare.logic — QUAND LA MAINMISE D'UN MATCH SE MONTRE : quand la mesure est possible (le
 * match a un rejeu, d'où vient le score) ou déjà servie. Zéro MESURÉ est une mesure : le bloc existe.
 */
import type { LeadShareValue } from '@/lib/charts/leadShare/model'

export interface LeadShareData {
  /** `header.lead_share` : fraction 0..1, absente = non mesuré. */
  leadShare?: LeadShareValue
  /** `header.lead_changes` : retournements, absents avec la mesure. */
  leadChanges?: number | null
  /** `header.replay_available`. */
  replayAvailable: boolean
}

export function showsLeadShare({ leadShare, replayAvailable }: LeadShareData): boolean {
  return leadShare != null || replayAvailable
}
