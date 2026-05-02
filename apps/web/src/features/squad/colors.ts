/**
 * colors.ts — couleurs partagées de la feature Escouade.
 *
 * Source unique des couleurs des joueurs sur la page squad : la pill du
 * joueur actif (gauche du multiselect, token `compare-a`) + les 3 slots
 * coéquipiers du `GamertagCombobox` (`SQUAD_TEAMMATE_COLOR_TOKENS`).
 *
 * Tous les charts qui rendent les joueurs côte à côte (per-minute, etc.)
 * doivent passer par `getSquadPlayerColors` pour conserver la cohérence
 * visuelle avec la pill et le combobox.
 */
import { getSeriesColors, type SemanticToken } from '@/lib/accessibility'

/** Token couleur du joueur actif (pill à gauche du multiselect). */
export const SQUAD_MAIN_PLAYER_TOKEN: SemanticToken = 'compare-a'

/**
 * Tokens couleur des 3 slots coéquipiers (ordre = ordre d'affichage dans
 * GamertagCombobox). Doit rester aligné avec `CHART_COLORS` dans
 * SquadLayout.tsx.
 */
export const SQUAD_TEAMMATE_COLOR_TOKENS: SemanticToken[] = [
  'narrative-dominant',
  'perf-tier-3',
  'divergent-pos',
]

/**
 * Retourne les couleurs hex résolues (palette active) pour les coéquipiers.
 * Aligné sur le pattern utilisé par SquadLayout (`CHART_COLORS`).
 */
export function getSquadTeammateColors(maxSlots = 3): string[] {
  return getSeriesColors(maxSlots, SQUAD_TEAMMATE_COLOR_TOKENS)
}

/**
 * Construit un mapping `gamertag → couleur hex` pour le main player + les
 * coéquipiers sélectionnés. Cohérent avec :
 *   - la pill `compare-a` du joueur actif
 *   - les tokens `narrative-dominant / perf-tier-3 / divergent-pos`
 *     attribués dans l'ordre par le `GamertagCombobox`.
 */
export function getSquadPlayerColors(
  mainGamertag: string,
  teammateGamertags: string[],
): Record<string, string> {
  const mainColor = getSeriesColors(1, [SQUAD_MAIN_PLAYER_TOKEN])[0]
  const teammateColors = getSquadTeammateColors(SQUAD_TEAMMATE_COLOR_TOKENS.length)
  const out: Record<string, string> = {}
  if (mainGamertag) out[mainGamertag] = mainColor
  teammateGamertags.forEach((gt, idx) => {
    out[gt] = teammateColors[idx % teammateColors.length]
  })
  return out
}
