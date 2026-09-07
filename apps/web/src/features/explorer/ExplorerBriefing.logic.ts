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
