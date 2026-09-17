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
 * Nombre de lignes que le bloc « Arme favorite » peut prendre sans faire grandir la
 * rangée « Par… », d'après la place LIBRE sous la cellule la plus haute.
 *
 * La hauteur se DÉCIDE, elle ne se mesure pas : le nombre de lignes de chaque cellule
 * est une donnée que le composant connaît déjà — une carte de dimension en a autant que
 * d'entrées, « Par contexte » en a deux, le Classement autant de chaînes AFFICHÉES (zéro
 * quand la capability du titre l'omet). Aucune lecture du DOM ici, et aucune permise
 * ailleurs : un écart constaté au gate visuel se corrige DANS cette formule.
 *
 * Le résultat choisit la FORME du bloc, jamais sa présence :
 *   - 2 → deux armes avec leur barre, la rangée ne bouge pas ;
 *   - 1 → une arme avec sa barre, la rangée ne bouge pas ;
 *   - 0 → forme compacte d'une seule ligne, la rangée gagne UNE ligne, jamais plus.
 *
 * Le plancher de deux lignes est celui de « Par contexte », la cellule la plus courte
 * de la rangée : en dessous, le bloc n'a de toute façon aucune place gratuite.
 */
export function favoriteWeaponSlots({
  dimensionLines,
  rankedLines,
}: {
  dimensionLines: number[]
  rankedLines: number
}): 0 | 1 | 2 {
  const rowLines = Math.max(2, rankedLines, ...dimensionLines)
  const free = rowLines - 2
  if (free >= 2) return 2
  return free === 1 ? 1 : 0
}
