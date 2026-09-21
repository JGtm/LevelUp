/**
 * barStackedGroups — LES GROUPES DE COLONNES D'UNE BARRE EMPILÉE VERTICALE.
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-21, décision D18). Le contrôle des armes spéciales range
 * ses socles en PUISSANCE puis TERRAIN sur UNE SEULE échelle : c'est la comparaison des
 * hauteurs entre les deux groupes qui porte la lecture, donc deux graphes côte à côte étaient
 * exclus. Il fallait un seul graphe qui dise quand même où finit un groupe et où commence
 * l'autre — SANS bandeau gris au-dessus (l'utilisateur l'a écarté) : un TRAIT VERTICAL discret
 * entre les deux colonnes de la frontière, et le nom du groupe avec son sous-total CENTRÉ EN
 * HAUT DANS le graphe, dans sa moitié.
 *
 * DEUX MÉCANISMES, ET CHACUN POUR SA RAISON :
 *   - le TRAIT est une `markLine` posée sur la première série, à l'index FRACTIONNAIRE de la
 *     frontière (`span - 0.5`) : elle vit dans le repère de la grille, donc elle tombe
 *     exactement dans la saignée entre deux colonnes quelle que soit la largeur rendue ;
 *   - les TITRES sont des textes `graphic`, positionnés en POURCENTAGE de la toile (même
 *     mécanisme que les titres de panneau de `squadIntensityProfileChart`). Un `graphic` ne
 *     connaît pas le repère de la grille : on centre chaque titre dans la bande de tracé
 *     ESTIMÉE ci-dessous. L'écart possible est de quelques pixels sur un titre centré — il ne
 *     se voit pas, là où un trait décalé d'autant couperait une colonne.
 *
 * Pur : aucun React, aucune couleur en dur — l'encre arrive de l'appelant (thème).
 */

/** Un groupe de colonnes : son titre (déjà localisé) et le nombre de colonnes qu'il couvre. */
export interface CategoryGroup {
  label: string
  span: number
}

/**
 * LA BANDE DE TRACÉ ESTIMÉE, en fraction de la largeur de la toile. La grille de
 * `buildBarStackedOption` ouvre à 8 px à gauche et 16 px à droite, puis `containLabel` y
 * ajoute la place des étiquettes de l'axe des valeurs — une largeur que seul le moteur connaît
 * au rendu. Ces deux bornes sont la moyenne observée sur les tailles de carte du dépôt.
 */
const PLOT_BAND_LEFT = 0.1
const PLOT_BAND_RIGHT = 0.97

/** Hauteur réservée aux titres de groupe, en pixels depuis le haut de la toile. */
const GROUP_TITLE_TOP = 2

/** Les frontières internes, en index fractionnaire de l'axe des catégories. */
export function groupBoundaries(groups: readonly CategoryGroup[]): number[] {
  const bornes: number[] = []
  let cumul = 0
  for (const groupe of groups.slice(0, -1)) {
    cumul += groupe.span
    bornes.push(cumul - 0.5)
  }
  return bornes
}

/**
 * La `markLine` des frontières — un trait par frontière interne, muet et sans étiquette.
 * Retourne `undefined` quand il n'y a qu'un groupe : un trait au bord ne sépare rien.
 */
export function groupSeparatorMarkLine(groups: readonly CategoryGroup[], color: string) {
  const bornes = groupBoundaries(groups)
  if (bornes.length === 0) return undefined
  return {
    silent: true,
    symbol: 'none' as const,
    label: { show: false },
    lineStyle: { color, type: 'solid' as const, width: 1 },
    data: bornes.map((x) => ({ xAxis: x })),
  }
}

/**
 * Le `graphic` des titres — un texte centré par groupe, en haut de la toile.
 * Retourne `undefined` quand aucun groupe n'est nommé (l'appelant peut vouloir le seul trait).
 */
export function groupTitleGraphic(groups: readonly CategoryGroup[], color: string) {
  const colonnes = groups.reduce((sum, g) => sum + g.span, 0)
  if (colonnes === 0 || groups.every((g) => g.label === '')) return undefined
  const largeur = PLOT_BAND_RIGHT - PLOT_BAND_LEFT
  let debut = 0
  const children = groups.map((groupe) => {
    const centre = PLOT_BAND_LEFT + ((debut + groupe.span / 2) / colonnes) * largeur
    debut += groupe.span
    return {
      type: 'text' as const,
      left: `${(centre * 100).toFixed(2)}%`,
      top: GROUP_TITLE_TOP,
      silent: true,
      style: { text: groupe.label, fill: color, fontSize: 11, align: 'center' as const },
    }
  })
  return { type: 'group' as const, children }
}
