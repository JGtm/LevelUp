/**
 * colors.ts — couleurs partagées de la feature Escouade.
 *
 * SOURCE UNIQUE de l'attribution des couleurs de joueurs sur la page Escouade :
 * l'ordre de la SÉLECTION. Le joueur de la page prend `squad-player-1` (pill à
 * gauche du multiselect), ses coéquipiers prennent `squad-player-2..4` dans
 * l'ordre des slots du `GamertagCombobox`. Aucun onglet n'attribue une couleur
 * selon l'ordre d'un bloc servi (fiches, nuage des vies, objectif) : un joueur
 * garde la même teinte d'un onglet à l'autre. Les onglets lisent la palette par
 * `useSquadPlayerPalette` (contexte de la page) ; garde-rail :
 * `playerColors.guard.test.ts`.
 *
 * La famille `squad-player-1..4` est dédiée à l'identité joueur : elle n'est
 * empruntée à aucun autre rôle sémantique et ses quatre valeurs sont
 * verrouillées par `lib/accessibility/squadPlayerTokens.test.ts` (contraste et
 * écart perceptuel, daltonisme compris).
 */
import { getSeriesColors, tokenCssVar, type SemanticToken } from '@/lib/accessibility'

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
 * Nombre maximum de coéquipiers sélectionnables en plus du joueur actif : exactement le
 * nombre de slots de couleur coéquipier (un 4e coéquipier n'aurait pas de teinte propre).
 * La barre de filtres (plafond du combobox) et le layout (init depuis les amis) le lisent.
 */
export const MAX_SELECTION = SQUAD_TEAMMATE_COLOR_TOKENS.length

/** Encre d'un joueur absent de la sélection (aucune identité de couleur). */
const UNSELECTED_PLAYER_INK = 'var(--muted-foreground)'

/**
 * Retourne les couleurs hex résolues (palette active) des slots coéquipiers, dans l'ordre
 * des slots du combobox de sélection.
 */
export function getSquadTeammateColors(maxSlots = 3): string[] {
  return getSeriesColors(maxSlots, SQUAD_TEAMMATE_COLOR_TOKENS)
}

/**
 * L'ATTRIBUTION : gamertag (casse servie) → jeton, dans l'ordre de la sélection. Le joueur
 * principal d'abord ; un coéquipier qui le répète ne lui reprend pas sa couleur. Au-delà des
 * slots, les jetons coéquipiers bouclent (modulo).
 */
function squadPlayerTokenEntries(
  mainGamertag: string,
  teammateGamertags: readonly string[],
): [string, SemanticToken][] {
  const out: [string, SemanticToken][] = []
  if (mainGamertag) out.push([mainGamertag, SQUAD_MAIN_PLAYER_TOKEN])
  teammateGamertags.forEach((gt, idx) => {
    out.push([gt, SQUAD_TEAMMATE_COLOR_TOKENS[idx % SQUAD_TEAMMATE_COLOR_TOKENS.length]])
  })
  return out
}

/**
 * Construit un mapping `gamertag → couleur hex` pour le main player + les
 * coéquipiers sélectionnés, dans l'ordre de la sélection (même attribution que
 * `squadPlayerPalette`).
 */
export function getSquadPlayerColors(
  mainGamertag: string,
  teammateGamertags: readonly string[],
): Record<string, string> {
  const entries = squadPlayerTokenEntries(mainGamertag, teammateGamertags)
  const hexes = getSeriesColors(entries.length, entries.map(([, token]) => token))
  const out: Record<string, string> = {}
  entries.forEach(([gt], i) => {
    if (!(gt in out)) out[gt] = hexes[i]
  })
  return out
}

/** La palette des joueurs de l'Escouade (une sélection donnée). */
export interface SquadPlayerPalette {
  /** gamertag (casse servie) → hex résolu : `colorByPlayer` des graphes ECharts. */
  colorByPlayer: Record<string, string>
  /** Jeton d'un joueur, gamertag insensible à la casse ; `null` hors sélection. */
  tokenOf: (gamertag: string) => SemanticToken | null
  /** Encre CSS (`var(--…)`) d'un joueur ; hors sélection, l'encre neutre. */
  inkOf: (gamertag: string) => string
}

/**
 * La palette d'une sélection : joueur de la page puis coéquipiers dans l'ordre du combobox.
 * Les onglets de l'Escouade la lisent par `useSquadPlayerPalette`.
 */
export function squadPlayerPalette(
  mainGamertag: string,
  teammateGamertags: readonly string[],
): SquadPlayerPalette {
  const byKey = new Map<string, SemanticToken>()
  for (const [gt, token] of squadPlayerTokenEntries(mainGamertag, teammateGamertags)) {
    const key = gt.toLowerCase()
    if (!byKey.has(key)) byKey.set(key, token)
  }
  const tokenOf = (gamertag: string) => byKey.get(gamertag.toLowerCase()) ?? null
  return {
    colorByPlayer: getSquadPlayerColors(mainGamertag, teammateGamertags),
    tokenOf,
    inkOf: (gamertag) => {
      const token = tokenOf(gamertag)
      return token ? tokenCssVar(token) : UNSELECTED_PLAYER_INK
    },
  }
}
