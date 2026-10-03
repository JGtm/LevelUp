/**
 * scales.ts — LES ÉCHELLES des formes qui n'en tirent pas une de la primitive
 * partagée.
 *
 * Séparées des composants qui les emploient (règle
 * `react-refresh/only-export-components`) et testables sans rendu : une échelle
 * fausse ne se voit pas sur une capture, elle se voit sur un test.
 */

/** Une ligne de la forme « écart à la parité », réduite à ce qui borne l'axe. */
export interface EcartScaleRow {
  pct: number | null
  parity: number
}

/**
 * L'AMPLITUDE COMMUNE d'une forme « écart » : le palier de cinq points au-dessus
 * du plus grand écart de ses lignes, six points au minimum.
 *
 * COMMUNE, et c'est le point : deux lignes de la même carte doivent se comparer
 * à l'oeil. Une amplitude par ligne ferait paraître un écart de deux points
 * aussi long qu'un écart de trente.
 */
export function ecartAmplitude(rows: EcartScaleRow[]): number {
  const gaps = rows.filter((r) => r.pct != null).map((r) => Math.abs((r.pct as number) - r.parity))
  return Math.max(6, Math.ceil(Math.max(6, ...gaps) / 5) * 5)
}
