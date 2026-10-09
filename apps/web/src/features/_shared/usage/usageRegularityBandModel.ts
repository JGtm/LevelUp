/**
 * usageRegularityBandModel.ts — LE MODÈLE DE LA BANDE DE RÉGULARITÉ : une case par match, teintée
 * par l'écart à la parité (au-dessus, au niveau, en dessous) ou laissée grise quand le match n'est
 * pas mesuré. Produit par la carte « Appui reçu » de Sessions (`session-detail/coordinationModel.ts`),
 * rendu par `UsageRegularityBand`.
 */

type UsageBandTone = 'above' | 'near' | 'below' | 'unmeasured'

export interface UsageBandCell {
  matchId: string
  tone: UsageBandTone
  tooltip: string
}
