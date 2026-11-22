/**
 * occupantTitles.ts : LES TITRES D'ANTAGONISTE SUR LE NOM D'UN OCCUPANT DE VÉHICULE.
 *
 * Un joueur embarqué n'a plus de pion : le calque des véhicules écrit son nom, à l'écran (noms empilés) et sur
 * la flèche hors cadre. Ce nom porte les mêmes titres que celui d'un pion, avec la même mise en page
 * (`replayLabels`, `offscreenChevron`) et la même pulsation (`antagonistMarks`) : ce fichier ne fait que choisir
 * l'occupant qui les porte et appeler ces dessins.
 */
import type { XY } from '../../../lib/replay/replayLogic'
import { drawTitlePulse, labelStyleOf, occupantTitleDraw, type AntagonistMarks } from './antagonistMarks'
import { drawOffscreenLabel } from './offscreenChevron'
import { drawNameLabel } from './replayLabels'

/** Un occupant : son slot, et la clé du film que porte son épisode quand le document la dit. */
export interface Occupant {
  slot: number
  xuid?: string
}

/** Ce que l'étiquette d'un occupant emprunte au calque : densité, contour, image lue, encre et titres. */
export interface OccupantLabel {
  color: string
  k: number
  labelStroke: string
  frame: number
  antagonists?: AntagonistMarks
}

/**
 * drawOccupantName écrit le nom d'un occupant sous le véhicule (`edge` : le bord au-delà duquel il se pose), avec
 * ses titres devant et leurs anneaux autour du véhicule quand il vient de prendre un titre.
 */
export function drawOccupantName(
  ctx: CanvasRenderingContext2D,
  c: XY,
  edge: number,
  occupant: Occupant & { name: string },
  label: OccupantLabel,
): void {
  const titles = occupantTitleDraw(label.antagonists, occupant, label, label.color)
  drawTitlePulse(ctx, c, titles, edge)
  drawNameLabel(ctx, c, occupant.name, labelStyleOf(label, titles), label.color, edge)
}

/**
 * namedOccupant : l'occupant dont le nom est écrit sur l'étiquette d'un véhicule hors cadre, le seul ou le
 * conducteur (siège 0) ; `undefined` quand aucun nom n'y est écrit (le repli « N joueurs » n'a pas de titres).
 */
export function namedOccupant<R extends Occupant & { seat?: number }>(
  rides: readonly R[],
  nameOf: (ride: R) => string | null,
): R | undefined {
  const named = rides.length === 1 ? rides[0] : rides.find((r) => r.seat === 0)
  return named && nameOf(named) ? named : undefined
}

/** drawOccupantOffscreenLabel écrit l'étiquette de la flèche hors cadre, les titres de `owner` devant le texte. */
export function drawOccupantOffscreenLabel(
  ctx: CanvasRenderingContext2D,
  mark: { at: XY; angle: number },
  text: string,
  owner: Occupant | undefined,
  label: OccupantLabel,
): void {
  const titles = occupantTitleDraw(label.antagonists, owner, label, label.color)
  drawOffscreenLabel(ctx, mark.at, mark.angle, text, labelStyleOf(label, titles), label.color)
}
