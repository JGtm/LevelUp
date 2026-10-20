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
