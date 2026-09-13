/**
 * access.ts — LES ACCÈS DE BASE à la matière du bloc « formes retenues »
 * (artefact 2ec1b8eb) : les cinq gestes d'équipement, le camp d'un joueur, les
 * deux effectifs d'un match, et la valeur d'un axe pour un joueur.
 *
 * POURQUOI UNE COUCHE D'ACCÈS. Les dix-neuf cartes posent la même question sur
 * quatre dénominateurs (mon équipe, le lobby, l'escouade seule, les occupations
 * de socle). Tout ce qui est commun aux quatre vit ici ; les agrégats vivent
 * dans `aggregates.ts`, les socles dans `pads.ts`, l'objectif dans
 * `objectives.ts`. Aucun React, aucune langue, aucune couleur.
 *
 * LES GRENADES N'Y SONT PAS, et le répulseur non plus : ce ne sont pas des
 * équipements pour les premières, et aucun canal ne mesure l'usage du second —
 * une ligne à zéro s'y lirait comme « jamais utilisé » quand la vérité est
 * « non mesuré ».
 */
import type {
  SquadFormesBlock,
  SquadFormesLobbyPlayer,
  SquadFormesMatch,
} from '@/lib/api/types'

/** Les cinq gestes d'équipement mesurés, DANS L'ORDRE DE L'ARTEFACT. */
export const EQUIPMENT_AXES = ['camo', 'wall', 'overshield', 'grapple', 'dropped'] as const

export type EquipmentAxis = (typeof EQUIPMENT_AXES)[number]

/**
 * L'axe des prises de socle, traité comme un geste de plus par les formes du
 * bloc 2 (même grammaire, autre dénominateur).
 */
export const PAD_AXIS = 'pad_pickups' as const

export type ShareAxis = EquipmentAxis | typeof PAD_AXIS

/** La valeur d'un axe pour une ligne de lobby. */
export function axisValue(player: SquadFormesLobbyPlayer, axis: ShareAxis): number {
  switch (axis) {
    case 'camo':
      return player.camo
    case 'wall':
      return player.wall
    case 'overshield':
      return player.overshield
    case 'grapple':
      return player.grapple
    case 'dropped':
      return player.dropped
    case PAD_AXIS:
      return player.pad_pickups
  }
}

/** Les matchs MESURÉS du scope (ceux qui portent un film décodé). */
export function measuredMatches(block: SquadFormesBlock): SquadFormesMatch[] {
  return (block.matches ?? []).filter((m) => m.measured)
}

/** Tous les matchs du scope, dans l'ordre de la page (mesurés ou non). */
export function allMatches(block: SquadFormesBlock): SquadFormesMatch[] {
  return block.matches ?? []
}

/** Les lignes de lobby d'un match (vide sur un match non mesuré). */
export function lobbyOf(match: SquadFormesMatch): SquadFormesLobbyPlayer[] {
  return match.lobby ?? []
}

/** La ligne du joueur dans un match, ou `undefined` s'il n'y est pas mesuré. */
export function playerRow(
  match: SquadFormesMatch,
  xuid: string,
): SquadFormesLobbyPlayer | undefined {
  return lobbyOf(match).find((p) => p.xuid === xuid)
}

/** La valeur d'un axe pour un joueur d'un match (0 s'il n'a pas de ligne). */
export function playerAxisValue(match: SquadFormesMatch, xuid: string, axis: ShareAxis): number {
  const row = playerRow(match, xuid)
  return row ? axisValue(row, axis) : 0
}

/** Le joueur appartient-il au camp du joueur de la page ? */
export function isMySide(match: SquadFormesMatch, player: SquadFormesLobbyPlayer): boolean {
  return match.player_team != null && player.team_id === match.player_team
}

/**
 * L'EFFECTIF DE MON CAMP et celui du lobby — les deux dénominateurs des parités.
 *
 * La source est l'effectif des PARTICIPANTS présents à la fin (bots compris),
 * jamais le nombre de lignes mesurées : un bot occupe une place du lobby et
 * abaisse la parité de tout le monde, même s'il n'a aucun geste au film. Repli
 * sur les lignes mesurées quand le match ne porte pas ses effectifs — mieux
 * vaut une parité approchée qu'aucune.
 */
export function matchSizes(match: SquadFormesMatch): { team: number; lobby: number } {
  const rows = lobbyOf(match)
  const lobby = match.lobby_size && match.lobby_size > 0 ? match.lobby_size : rows.length
  const team =
    match.team_size && match.team_size > 0
      ? match.team_size
      : rows.filter((p) => isMySide(match, p)).length
  return { team, lobby }
}

/** La parité d'un effectif : la part d'un joueur si tous en faisaient autant. */
export function parityOf(size: number): number | null {
  return size > 0 ? 100 / size : null
}

/** Une part en pourcentage, ou `null` quand le dénominateur est nul (NON MESURÉ). */
export function sharePct(numerator: number, denominator: number): number | null {
  return denominator > 0 ? (numerator / denominator) * 100 : null
}

/** La moyenne d'une série, ou `null` si elle est vide. */
export function average(values: number[]): number | null {
  if (values.length === 0) return null
  return values.reduce((a, v) => a + v, 0) / values.length
}
