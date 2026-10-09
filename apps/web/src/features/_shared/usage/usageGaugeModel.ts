/**
 * usageGaugeModel.ts — LE MODÈLE DE LA JAUGE « écart à la parité » : sa valeur, son trait, ses
 * textes ; et la ligne de jauges qui les porte. Produit par la carte « Appui reçu » de Sessions
 * (`session-detail/coordinationModel.ts`), rendu par `UsageGaugeGrid`.
 *
 * `null` N'EST PAS ZÉRO : une part sans dénominateur rend une jauge vide au tiret, jamais un 0 %.
 */

/** Une jauge 0..100 % : sa valeur, son trait (parité ou habituel), son texte et son infobulle. */
export interface UsageGaugeModel {
  key: string
  valuePct: number | null
  parityPct: number | null
  valueText: string
  tooltip: string
}

/** Une ligne de jauges : la grandeur et une jauge par colonne de la grille. */
export interface UsageGaugeRowModel {
  key: string
  label: string
  gauges: UsageGaugeModel[]
}
