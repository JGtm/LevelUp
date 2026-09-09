/**
 * blockRowPacking — répartit une SUITE ORDONNÉE de blocs de catalogue (catégories de
 * médailles, familles de citations) sur des rangées, en donnant à chaque bloc une
 * largeur proportionnelle à SON CONTENU.
 *
 * Le problème (retour utilisateur 2026-09-09) : les pages Médailles et Citations
 * empilaient un bloc pleine largeur par catégorie, quel que soit son contenu. Une
 * catégorie de 2 médailles occupait la même bande de 1 300 px qu'une catégorie de 25 —
 * l'écran se lisait comme une colonne de vides. « Capture du drapeau » et « Roi de la
 * colline » tenaient largement côte à côte, et parfois à trois.
 *
 * La règle est un simple comptage : combien de vignettes le bloc a-t-il à montrer ?
 *   - 3 ou moins       → tiers de largeur (une rangée de vignettes y tient)
 *   - de 4 à 8         → demi-largeur (deux rangées au plus)
 *   - 9 et plus        → pleine largeur
 *
 * L'ORDRE EST PRÉSERVÉ : le tri choisi dans la barre d'outils (total de la catégorie,
 * nombre par médaille, nom) reste la seule chose qui ordonne les blocs. Le packing est
 * glouton — il ferme une rangée dès que le bloc suivant n'y tiendrait plus — et ne
 * réordonne jamais pour mieux remplir : une grille qui se réarrange à chaque filtre
 * serait illisible.
 *
 * La dernière rangée peut rester incomplète : `rowRemainder` donne le nombre de colonnes
 * libres, que la vue rend en colonne fantôme. On ne dilate PAS les blocs pour combler —
 * une catégorie de 2 médailles étirée sur toute la largeur, c'est exactement le défaut
 * qu'on corrige.
 */

/** Colonnes de la grille de référence. 6 = PPCM des trois largeurs (2, 3, 6). */
export const BLOCK_GRID_COLUMNS = 6

/** Seuils de comptage (inclusifs) au-delà desquels le bloc s'élargit. */
const SPAN_THRESHOLDS: ReadonlyArray<{ maxItems: number; span: number }> = [
  { maxItems: 3, span: 2 },
  { maxItems: 8, span: 3 },
]

/**
 * Largeur d'un bloc, en colonnes de `BLOCK_GRID_COLUMNS`, d'après son nombre de
 * vignettes. Un bloc vide garde la largeur minimale (il n'a rien à montrer).
 */
export function blockSpan(itemCount: number): number {
  const rule = SPAN_THRESHOLDS.find((r) => itemCount <= r.maxItems)
  return rule ? rule.span : BLOCK_GRID_COLUMNS
}

/** Une rangée : les blocs dans l'ordre d'origine, et la largeur de chacun. */
export interface PackedRow<T> {
  blocks: T[]
  spans: number[]
  /** Colonnes libres en fin de rangée (0 si la rangée est pleine). */
  rowRemainder: number
}

/**
 * Répartit `blocks` en rangées de `BLOCK_GRID_COLUMNS` colonnes au plus, sans jamais
 * changer leur ordre. `countOf` donne le nombre de vignettes d'un bloc.
 */
export function packBlockRows<T>(blocks: readonly T[], countOf: (block: T) => number): PackedRow<T>[] {
  const rows: PackedRow<T>[] = []
  let current: T[] = []
  let spans: number[] = []
  let used = 0

  const close = () => {
    if (current.length === 0) return
    rows.push({ blocks: current, spans, rowRemainder: BLOCK_GRID_COLUMNS - used })
    current = []
    spans = []
    used = 0
  }

  for (const block of blocks) {
    const span = blockSpan(countOf(block))
    if (used + span > BLOCK_GRID_COLUMNS) close()
    current.push(block)
    spans.push(span)
    used += span
  }
  close()
  return rows
}

/**
 * Valeur `grid-template-columns` d'une rangée : une piste par bloc, plus une piste
 * fantôme pour les colonnes libres de la dernière rangée.
 */
export function rowGridTemplate(row: PackedRow<unknown>): string {
  const tracks = row.spans.map((s) => `${s}fr`)
  if (row.rowRemainder > 0) tracks.push(`${row.rowRemainder}fr`)
  return tracks.join(' ')
}
