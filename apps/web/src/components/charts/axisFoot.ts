/**
 * axisFoot — le pied d'un graphe ECharts à bande sous l'axe X : étiquettes de l'axe principal,
 * bande (résultats, victoires / défaites), étiquettes de l'axe de la bande.
 *
 * Deux grilles superposées par graphe (courbes en haut, bande en bas) sur une même abscisse. Le
 * bas de la grille des courbes (`grid[0].bottom`) et la position de la bande (`grid[1].bottom`)
 * dépendent de la place que prennent les étiquettes : police, hauteur de ligne, marge, nombre de
 * lignes. Écrits à la main, ces nombres dérivaient dès qu'une étiquette changeait de taille et la
 * bande venait se poser SUR les dates. Ici ils se CALCULENT depuis la description des étiquettes,
 * et les mêmes valeurs (`margin`, `lineHeight`) sont écrites dans l'axe (`footLabelStyle`) : ce
 * qui est réservé est exactement ce qui est dessiné.
 *
 * Repère vertical : y = 0 est le bord bas de la grille des courbes, y croît vers le bas.
 *
 *   0 ─ marge + lignes de `above` ─ écart ─ bande ─ marge + lignes de `below` ─ marge de pied
 */

/** Un bloc d'étiquettes d'axe : `lines` lignes de `lineHeight` px, à `margin` px de l'axe. */
export interface FootLabels {
  lines: number
  lineHeight: number
  margin: number
}

export interface AxisFootSpec {
  /** Étiquettes de l'axe des courbes, sous le bord bas de leur grille ; null = aucune. */
  above: FootLabels | null
  /** La bande ; `gap` = espace entre le bloc `above` (ou le bord de la grille) et son bord haut. */
  band: { height: number; gap: number } | null
  /** Étiquettes de l'axe de la bande, sous elle ; null = aucune. */
  below: FootLabels | null
  /** Marge au pied du graphe, sous le dernier bloc. */
  pad: number
}

export interface AxisFootLayout {
  /** `grid[0].bottom` : tout le pied. */
  gridBottom: number
  /** `grid[1].bottom` ; 0 sans bande. */
  bandBottom: number
  /** `grid[1].height` ; 0 sans bande. */
  bandHeight: number
}

/** Hauteur de ligne dérivée de la police : 1,3 × la taille, arrondie. */
export function lineHeightFor(fontSize: number): number {
  return Math.round(fontSize * 1.3)
}

/** Hauteur d'un bloc d'étiquettes : sa marge puis ses lignes ; 0 sans bloc. */
export function labelsHeight(block: FootLabels | null): number {
  return block ? block.margin + block.lines * block.lineHeight : 0
}

/** Le pied d'un graphe : `gridBottom` et la position de la bande, calculés depuis `spec`. */
export function layoutAxisFoot(spec: AxisFootSpec): AxisFootLayout {
  const bandTop = labelsHeight(spec.above) + (spec.band ? spec.band.gap : 0)
  const bandEdge = bandTop + (spec.band ? spec.band.height : 0)
  const gridBottom = bandEdge + labelsHeight(spec.below) + spec.pad
  return {
    gridBottom,
    bandBottom: spec.band ? gridBottom - bandEdge : 0,
    bandHeight: spec.band ? spec.band.height : 0,
  }
}

/** Le style d'axe qui dessine ce que `layoutAxisFoot` a réservé : même marge, même hauteur de ligne. */
export function footLabelStyle(block: FootLabels) {
  return { interval: 0, margin: block.margin, lineHeight: block.lineHeight }
}
