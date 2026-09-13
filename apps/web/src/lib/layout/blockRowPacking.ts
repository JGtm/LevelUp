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
 *   - 3 ou moins       → un tiers de la grille de référence
 *   - de 4 à 8         → une demie
 *   - 9 et plus        → la grille entière
 *
 * L'ORDRE EST PRÉSERVÉ : le tri choisi dans la barre d'outils (total de la catégorie,
 * nombre par médaille, nom) reste la seule chose qui ordonne les blocs. Le packing est
 * glouton — il ferme une rangée dès que le bloc suivant n'y tiendrait plus — et ne
 * réordonne jamais pour mieux remplir : une grille qui se réarrange à chaque filtre
 * serait illisible.
 *
 * UNE RANGÉE OCCUPE TOUJOURS TOUTE LA LARGEUR (retour utilisateur 2026-09-13). Les
 * largeurs ci-dessus sont des PROPORTIONS, pas des tailles absolues : sur une rangée
 * incomplète, les pistes présentes se partagent toute la place au prorata de leurs
 * spans (un bloc seul = 100 % ; un « tiers » avec une « demie » = deux cinquièmes et
 * trois cinquièmes). La version précédente ajoutait une piste fantôme pour les colonnes
 * libres : un bloc seul restait à un tiers de rangée et laissait un trou à droite —
 * c'est ce défaut qui est corrigé ici. Le RAPPORT entre blocs d'une même rangée, lui,
 * reste dicté par le comptage.
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
    rows.push({ blocks: current, spans })
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
 * Valeur `grid-template-columns` d'une rangée : UNE piste par bloc, et rien d'autre.
 * Les unités `fr` se partagent tout l'espace disponible, donc une rangée incomplète est
 * dilatée jusqu'au bord au lieu de laisser un trou à droite.
 */
export function rowGridTemplate(row: PackedRow<unknown>): string {
  return row.spans.map((s) => `${s}fr`).join(' ')
}
