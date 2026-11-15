import type { ImprovementCampaign } from '@/lib/playerProfile'

/**
 * Un départ sans échantillon (`snapshot_sample` = 0) est un départ NON MESURÉ : aucun match mesuré
 * sur l'axe avant le démarrage, la valeur `snapshot_value` (0) n'est pas une mesure et ne doit ni
 * s'afficher comme telle ni servir de base à une progression chiffrée.
 */
export function isStartMeasured(campaign: Pick<ImprovementCampaign, 'snapshot_sample'>): boolean {
  return campaign.snapshot_sample > 0
}

/** Écart valeur courante lissée − départ ; indéfini sans valeur courante ou sans départ mesuré. */
export function campaignDelta(
  campaign: Pick<ImprovementCampaign, 'snapshot_sample' | 'snapshot_value' | 'current_value_lowess'>,
): number | undefined {
  if (!isStartMeasured(campaign) || campaign.current_value_lowess === undefined) return undefined
  return campaign.current_value_lowess - campaign.snapshot_value
}
