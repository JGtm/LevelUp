/**
 * ExplorerBriefing.logic — helpers purs du bandeau de briefing (mode Matchs).
 *
 * Aucune dépendance React/DOM : logique testable en isolation (formatage des
 * deltas signés, tokens de couleur). Les composants du bandeau consomment ces
 * helpers.
 */
import type { SemanticToken } from '@/lib/accessibility'

// formatSignedFixed (delta signé à N décimales, glyphe '−' U+2212) est centralisé
// dans `@/lib/formatters` — importé directement par les modules du briefing.
//
// formatSignedPoints et isFullHistoryScope ont DÉMÉNAGÉ dans `@/lib/baseline` le
// 2026-09-06 : la page Escouade en a besoin pour son écart d'échange « vs habituel »,
// et le ratchet d'imports croisés (plafond atteint) interdisait un squad -> explorer
// de plus. Une copie aurait donné deux définitions du même écart.

/** Signe d'un nombre : -1 / 0 / 1 (0 pour nul, absent ou non fini). */
export function signOf(v: number | null | undefined): -1 | 0 | 1 {
  if (v == null || !Number.isFinite(v) || v === 0) return 0
  return v > 0 ? 1 : -1
}

/**
 * Token de couleur d'un delta signé (positif = gagnant, négatif = perdant, nul =
 * neutre). Helper CANONIQUE : centralisé ici (CLAUDE.md §6, 3e usage avec la tuile
 * Classement V3) — ne JAMAIS ré-inliner ce ternaire dans un composant du briefing
 * (garde-rail `explorerDeltaToken.guard.test.ts`).
 */
export function deltaToken(v: number | null | undefined): SemanticToken {
  const s = signOf(v)
  return s > 0 ? 'outcome-win' : s < 0 ? 'outcome-loss' : 'outcome-draw'
}

/**
 * Hauteur, en lignes de liste, d'une carte « Par… » en état vide (`insufficient`) : l'état
 * vide compact (titre + une à deux lignes de texte dans une cellule de 240 px, marges
 * réduites) pèse environ deux lignes de liste. Valeur posée, ajustable au gate visuel (même
 * doctrine que favoriteWeaponSlots : la hauteur se décide, elle ne se mesure pas).
 */
export const INSUFFICIENT_CARD_LINES = 2

/** Nombre de lignes que pèse une carte de dimension dans la formule de hauteur de la rangée. */
export function dimensionCardLines(dim: {
  entries?: readonly unknown[] | null
  insufficient?: boolean
}): number {
  return dim.insufficient ? INSUFFICIENT_CARD_LINES : (dim.entries ?? []).length
}

/**
 * Nombre d'armes que le bloc « Arme favorite » montre : DEUX quand la rangée « Par… » a
 * la place de les porter sans grandir, UNE sinon. Jamais zéro — le bloc est toujours là,
 * et toujours dans sa carte.
 *
 * La hauteur se DÉCIDE, elle ne se mesure pas : le nombre de lignes de chaque cellule
 * est une donnée que le composant connaît déjà — une carte de dimension en a autant que
 * d'entrées, « Par contexte » en a deux, le Classement autant de chaînes AFFICHÉES (zéro
 * quand la capability du titre l'omet). Aucune lecture du DOM ici, et aucune permise
 * ailleurs : un écart constaté au gate visuel se corrige DANS cette formule.
 *
 * CE QUE LA FORMULE NE DÉCIDE PLUS. Au gate visuel du 2026-09-17 l'utilisateur a tranché
 * que le bloc porte la carte de ses voisines dans TOUS les cas : le rendu nu empilé « ne
 * collait pas du tout » avec le reste de la rangée. L'harmonie visuelle l'emporte donc sur
 * la promesse de hauteur constante, la forme compacte a disparu, et il ne reste ici qu'un
 * choix de NOMBRE d'armes.
 *
 * LA CONSTANTE DE BASE vaut 2 en cellule propre et 4 en empilé. Deux, c'est la hauteur
 * de « Par contexte », la cellule la plus courte de la rangée. Quatre, parce qu'empilé
 * SOUS cette carte le bloc paie en plus le chrome de sa propre carte et l'espacement de
 * la pile — environ deux lignes que la formule nue ne comptait pas. Une seconde arme n'est
 * servie que si la rangée offre deux lignes au-delà de cette base.
 */
export function favoriteWeaponSlots({
  dimensionLines,
  rankedLines,
  stacked,
}: {
  dimensionLines: number[]
  rankedLines: number
  // stacked : le bloc est monté sous « Par contexte », dans la même cellule.
  stacked: boolean
}): 1 | 2 {
  const base = stacked ? 4 : 2
  const rowLines = Math.max(base, rankedLines, ...dimensionLines)
  return rowLines - base >= 2 ? 2 : 1
}

/**
 * Largeur minimale d'une tuile du socle (px) : en dessous, une configuration ne « tient » pas.
 * 150 px laisse les 8 tuiles sur une ligne dans la largeur maximale du contenu de page (1 256 px
 * requis, 1 272 px disponibles).
 */
export const SOCLE_TILE_MIN_PX = 150
/** Gouttière entre tuiles du socle (px) : `gap-2`. */
export const SOCLE_GAP_PX = 8

/**
 * Classe de grille du socle de tuiles, selon le nombre de tuiles rendues (1..8).
 *
 * Règle (colonnes égales, alignées d'une ligne à l'autre) : si toutes les tuiles tiennent sur
 * une ligne, une ligne pleine largeur ; sinon des lignes ÉQUILIBRÉES (8 -> 4+4, 7 -> 4+3,
 * 6 -> 3+3, 5 -> 3+2), et à défaut deux colonnes (plancher, téléphone). Les colonnes
 * candidates d'un socle de n tuiles sont n, ceil(n/2) et 2, sans doublon, jamais sous 2 (n si
 * n < 2) : pas de palier intermédiaire, car une colonne de plus que ceil(n/2) laisse une tuile seule ou des
 * lignes déséquilibrées (7 -> 3+3+1, 8 -> 3+3+2). Une configuration à c colonnes tient dès que
 * la largeur du conteneur vaut au moins c * SOCLE_TILE_MIN_PX + (c - 1) * SOCLE_GAP_PX. La
 * base est la plus petite candidate, chaque palier `@min-[Npx]` en ajoute une, du plus étroit
 * au plus large.
 *
 * Les classes sont LITTÉRALES : Tailwind ne génère que ce qu'il lit dans le source, une
 * concaténation `@min-[${x}px]` ne produirait rien. Le test de la table recalcule les seuils
 * depuis la règle et refuse toute dérive. Le palier se mesure sur le conteneur
 * (`@container`), pas sur la fenêtre : le bandeau peut être plus étroit qu'elle.
 */
export const SOCLE_GRID_CLASSES: Readonly<Record<number, string>> = {
  1: 'grid-cols-1',
  2: 'grid-cols-2',
  3: 'grid-cols-2 @min-[466px]:grid-cols-3',
  4: 'grid-cols-2 @min-[624px]:grid-cols-4',
  5: 'grid-cols-2 @min-[466px]:grid-cols-3 @min-[782px]:grid-cols-5',
  6: 'grid-cols-2 @min-[466px]:grid-cols-3 @min-[940px]:grid-cols-6',
  7: 'grid-cols-2 @min-[624px]:grid-cols-4 @min-[1098px]:grid-cols-7',
  8: 'grid-cols-2 @min-[624px]:grid-cols-4 @min-[1256px]:grid-cols-8',
}

/** Classe de grille du socle pour `tileCount` tuiles ; borné à 1..8 (plafond du socle). */
export function socleGridClass(tileCount: number): string {
  const n = Math.min(8, Math.max(1, Math.trunc(tileCount) || 1))
  return SOCLE_GRID_CLASSES[n]
}
