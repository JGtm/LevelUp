/**
 * valueGridModel.ts — LA PROJECTION D'UNE GRILLE DE VALEURS : lignes = individus, colonnes =
 * grandeurs, CHAQUE COLONNE AVEC SA PROPRE ÉCHELLE.
 *
 * POURQUOI CETTE FORME, ET POURQUOI SA LOGIQUE EST ICI. Deux blocs de la page match posent
 * exactement la même question — « qui a fait combien de quoi » — sur deux jeux de grandeurs
 * différents : le bilan d'équipement du rejeu et les actions d'objectif du scoreboard. La
 * réponse tenait dans deux tableaux de chiffres, que l'utilisateur a jugés hors style
 * (2026-09-03) ; elle tient maintenant dans une grille de barres. Deux features distinctes
 * l'emploient (`match-replay` et `match-view`) : la primitive vit donc dans `components/`,
 * jamais dans l'une des deux — le ratchet `tools/lint-cross-feature-imports.mjs` interdit
 * l'import croisé, et il a raison.
 *
 * CHAQUE COLONNE A SON ÉCHELLE, et c'est le sujet. Un mur posé se compare à un mur, jamais à
 * une grenade lancée : une échelle commune écraserait toutes les colonnes rares contre le
 * zéro. En contrepartie l'écran DOIT graduer chaque colonne (0 · milieu · borne), sans quoi
 * deux barres de même longueur dans deux colonnes se liraient comme deux valeurs égales.
 *
 * L'ORDRE DES LIGNES EST FIXE ET GROUPÉ. Les lignes gardent l'ordre que l'appelant leur donne
 * (équipe par équipe) d'une colonne à l'autre : un individu se lit EN LIGNE. Un filet sépare
 * deux groupes consécutifs — `separators` porte les index de ligne qu'il précède.
 *
 * `null` N'EST PAS ZÉRO : une valeur nulle est une grandeur NON MESURÉE (la jointure n'a pas
 * pu être tentée). Sa barre est vide et sa cellule écrit le repli de l'appelant — elle
 * n'entre ni dans le maximum de la colonne ni dans son total. Un zéro mesuré, lui, est une
 * mesure et compte comme telle.
 *
 * Pur : aucun React, aucune couleur en dur, aucune langue — l'appelant fournit les libellés,
 * les encres (jetons résolus) et le formatage.
 */

/** Une ligne de la grille : un individu, son groupe, et son identité visuelle. */
export interface ValueGridRow {
  /** Clé de rendu, unique dans la grille. */
  key: string
  /** Nom affiché à gauche, aligné d'une colonne à l'autre. */
  label: string
  /**
   * GROUPE de la ligne (un camp, typiquement). Deux groupes consécutifs différents sont
   * séparés par un filet ; la valeur elle-même n'est jamais affichée.
   */
  group: string
  /** Encre du trait d'identité posé devant le nom (CSS). Absent = pas de trait. */
  accent?: string
  /** Ligne mise en avant (le joueur de la page) : nom en gras. */
  emphasis?: boolean
  /** Infobulle du nom (identité complète : nom + camp). */
  hint?: string
}

/** Une colonne demandée : sa clé, son nom, et la façon dont elle se lit. */
export interface ValueGridColumnInput {
  key: string
  label: string
  /**
   * La colonne porte une DURÉE : le milieu de son axe n'est pas arrondi à l'entier (une
   * demi-borne de 2:30 est une graduation juste, « 3 » ne l'est pas).
   */
  duration?: boolean
  /**
   * La colonne affiche son TOTAL en en-tête. Faux pour une colonne dont la somme n'a pas de
   * sens — un « meilleur temps » ne s'additionne pas (cf. `objectiveTeamTotal`, agrégat `max`).
   */
  showTotal?: boolean
}

/** Les entrées de la projection : les lignes, les colonnes, et quatre lectures. */
export interface ValueGridInput {
  rows: ValueGridRow[]
  columns: ValueGridColumnInput[]
  /** La valeur d'une cellule. `null` = NON MESURÉ (cf. en-tête), jamais un zéro déguisé. */
  value: (rowIndex: number, colIndex: number) => number | null
  /** Le texte d'une valeur pour cette colonne (entier, m:ss, …). */
  format: (value: number, colIndex: number) => string
  /** L'encre d'une barre (CSS — `tokenCssVar(...)` côté appelant, jamais un hex). */
  color: (rowIndex: number, colIndex: number) => string
  /** L'infobulle d'une barre, qui reçoit le texte déjà formaté de la valeur. */
  tooltip: (rowIndex: number, colIndex: number, text: string) => string
  /** Ce qu'écrit une cellule NON MESURÉE. Défaut : le tiret cadratin. */
  notMeasured?: string
}

/** Une cellule projetée : sa valeur, son texte, sa longueur relative, son encre. */
export interface ValueGridCell {
  value: number | null
  text: string
  /** Part de la borne de colonne, dans [0, 1]. Une valeur non mesurée vaut 0. */
  fraction: number
  color: string
  tooltip: string
}

/** Une colonne projetée : son en-tête, son total, et ses trois graduations. */
export interface ValueGridColumn {
  key: string
  label: string
  /** Total de colonne déjà formaté, ou `null` quand la colonne n'en porte pas. */
  totalText: string | null
  /** Borne haute de l'échelle de la colonne. */
  bound: number
  /** Graduations écrites du pied de colonne : zéro, milieu, borne. */
  axis: [string, string, string]
}

/** La grille projetée, prête à rendre. */
export interface ValueGridModel {
  rows: ValueGridRow[]
  columns: ValueGridColumn[]
  /** `cells[rowIndex][colIndex]`. */
  cells: ValueGridCell[][]
  /** Index des lignes précédées d'un filet de séparation de groupe. */
  separators: number[]
}

/** Le repli d'une cellule non mesurée, quand l'appelant n'en donne pas d'autre. */
const NOT_MEASURED = '—'

/**
 * valueGridBound — le palier « rond » au-dessus du maximum d'une colonne.
 *
 * Une borne égale au maximum collerait la plus grande barre au bord de son rail et rendrait
 * la graduation de fin illisible ; un palier rond donne en plus un milieu qui se lit (5, 10,
 * 30…). Les quatre régimes suivent l'ordre de grandeur : l'unité en dessous de 5, la paire
 * jusqu'à 12, le pas de 5 jusqu'à 60, la demi-minute au-delà (les durées de la page match
 * s'expriment en secondes).
 */
export function valueGridBound(max: number): number {
  if (!Number.isFinite(max) || max <= 0) return 1
  if (max <= 5) return Math.max(1, Math.ceil(max))
  if (max <= 12) return Math.ceil(max / 2) * 2
  if (max <= 60) return Math.ceil(max / 5) * 5
  return Math.ceil(max / 30) * 30
}

/** Le milieu écrit d'un axe : arrondi à l'entier, sauf sur une colonne de durée. */
function axisMid(bound: number, duration: boolean): number {
  return duration ? bound / 2 : Math.round(bound / 2)
}

/** buildValueGrid — la projection complète. Aucun effet de bord, aucune lecture d'horloge. */
export function buildValueGrid(input: ValueGridInput): ValueGridModel {
  const { rows, columns, value, format, color, tooltip } = input
  const notMeasured = input.notMeasured ?? NOT_MEASURED

  const projected = columns.map((col, colIndex) => {
    let max = 0
    let sum = 0
    for (let r = 0; r < rows.length; r += 1) {
      const v = value(r, colIndex)
      if (v == null) continue
      sum += v
      if (v > max) max = v
    }
    const bound = valueGridBound(max)
    return {
      key: col.key,
      label: col.label,
      totalText: col.showTotal ? format(sum, colIndex) : null,
      bound,
      axis: [
        format(0, colIndex),
        format(axisMid(bound, col.duration === true), colIndex),
        format(bound, colIndex),
      ] as [string, string, string],
    }
  })

  const cells = rows.map((_row, rowIndex) =>
    columns.map((_col, colIndex) => {
      const v = value(rowIndex, colIndex)
      const text = v == null ? notMeasured : format(v, colIndex)
      const bound = projected[colIndex].bound
      return {
        value: v,
        text,
        fraction: v == null ? 0 : Math.max(0, Math.min(1, v / bound)),
        color: color(rowIndex, colIndex),
        tooltip: tooltip(rowIndex, colIndex, text),
      }
    }),
  )

  const separators: number[] = []
  for (let r = 1; r < rows.length; r += 1) {
    if (rows[r].group !== rows[r - 1].group) separators.push(r)
  }

  return { rows, columns: projected, cells, separators }
}
