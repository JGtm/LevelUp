/**
 * followLogic.ts — OÙ LA CARTE SE CENTRE QUAND ON SUIT UN JOUEUR (« Suivre », lot 4 du plan
 * « maquettes du rejeu »). Logique pure (followLogic.test.ts) ; le câblage vit dans
 * `hooks/useReplayFollow.ts`.
 *
 * LA POSITION EST CELLE DU PORTEUR, À L'IMAGE LUE, FRACTIONNAIRE COMPRISE : un joueur à bord d'un
 * véhicule se suit sur le véhicule (le bipède ne réplique plus, sa position tenue serait fausse),
 * sinon sur son bipède, avec la fenêtre après-mort des autres lecteurs (`posOfPlayerAt`) — la
 * carte reste sur l'endroit de la mort un instant, puis plus rien.
 *
 * SANS POSITION, `null` : la carte reste où elle est. Ni saut vers le centre, ni position
 * devinée. Un joueur qui réapparaît ailleurs est une AUTRE vie : la carte le rejoint.
 *
 * LE JOUEUR EST UN `ReplayPlayer`, PAS UN XUID : un bot n'en a pas, ses vies suffisent. Seule la
 * lecture du véhicule passe par le xuid (l'épisode d'occupation nomme son occupant par xuid) ; un
 * bot embarqué n'y est donc pas retrouvé et se suit sur son bipède, comme les calques de porteur.
 */
import { posOfPlayerAt } from './livesPosition'
import type { EmbarkedPosAt } from './carrierPosition'
import type { XY, ZoomLevel } from '../../../lib/replay/replayLogic'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

/** Le palier de zoom MINIMAL de Suivre : un joueur se suit de près, pas sur la carte entière. */
export const FOLLOW_MIN_LEVEL: ZoomLevel = 2

/** La position du joueur à l'image lue (`null` = inconnue : la carte ne bouge pas). */
export function followPositionAt(
  player: ReplayPlayer,
  frame: number,
  embarkedAt: EmbarkedPosAt,
  deathFrames: number,
): XY | null {
  return embarkedAt(player.xuid, frame) ?? posOfPlayerAt(player.lives, frame, deathFrames)
}

/** Deux positions sont-elles le même endroit ? Sert à ne rien demander à la carte quand le joueur n'a pas bougé. */
export function sameSpot(a: XY | null, b: XY | null): boolean {
  if (a === null || b === null) return a === b
  return a.x === b.x && a.y === b.y
}
