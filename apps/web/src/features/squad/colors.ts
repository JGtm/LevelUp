/**
 * colors.ts — couleurs partagées de la feature Escouade.
 *
 * Source unique des couleurs des joueurs sur la page squad : la pill du
 * joueur actif (gauche du multiselect, token `squad-player-1`) + les 3 slots
 * coéquipiers du `GamertagCombobox` (`SQUAD_TEAMMATE_COLOR_TOKENS`).
 *
 * La famille `squad-player-1..4` est dédiée à l'identité joueur : elle n'est
 * empruntée à aucun autre rôle sémantique et ses quatre valeurs sont
 * verrouillées par `lib/accessibility/squadPlayerTokens.test.ts` (contraste et
 * écart perceptuel, daltonisme compris).
 *
 * Tous les charts qui rendent les joueurs côte à côte (per-minute, etc.)
 * doivent passer par `getSquadPlayerColors` pour conserver la cohérence
 * visuelle avec la pill et le combobox.
 */
import { getSeriesColors, type SemanticToken } from '@/lib/accessibility'

/** Token couleur du joueur actif (pill à gauche du multiselect). */
export const SQUAD_MAIN_PLAYER_TOKEN: SemanticToken = 'squad-player-1'

/**
 * Tokens couleur des 3 slots coéquipiers (ordre = ordre d'affichage dans
 * GamertagCombobox).
 */
export const SQUAD_TEAMMATE_COLOR_TOKENS: SemanticToken[] = [
  'squad-player-2',
  'squad-player-3',
  'squad-player-4',
]

/**
 * Nombre maximum de coéquipiers sélectionnables en plus du joueur actif.
 *
 * Défini ICI parce que c'est exactement le nombre de slots de couleur
 * coéquipier : un 4e coéquipier n'aurait pas de teinte propre. Vit dans un
 * module partagé (et non dans un `.tsx`) depuis l'extraction de
 * `SquadFilterBar` le 2026-09-20 — la barre et le layout en ont tous deux
 * besoin (plafond du combobox d'un côté, init depuis les amis de l'autre).
 */
export const MAX_SELECTION = SQUAD_TEAMMATE_COLOR_TOKENS.length

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
 *   - la pill `squad-player-1` du joueur actif
 *   - les tokens `squad-player-2 / -3 / -4` attribués dans l'ordre par le
 *     `GamertagCombobox`.
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
