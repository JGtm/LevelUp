/**
 * markerOutline.ts — LE CERNE DEUX TONS DU PION et la géométrie qui en découle : une bande sombre
 * contre le noyau, bordée d'un filet clair.
 *
 * POURQUOI DEUX TONS. Le contour d'avant était l'encre de mise en page
 * (`--muted-foreground`) à 62 % d'opacité : mélangé à 38 % du sol, il n'atteignait jamais 3:1
 * contre le fond, sur aucune des 1 280 combinaisons de palette, de thème, de teinte et de vision
 * mesurées. Un seul ton ne peut pas tenir un fond qui va du presque noir au presque blanc ; deux le
 * peuvent : la bande sombre se lit contre un sol clair, le filet clair contre un sol sombre, et le
 * meilleur des deux atteint au moins 4,4:1 partout (garde-fou : `mapReliefTokens.test.ts`). Les
 * deux encres sont des jetons (`marker-outline-band`, `marker-outline-rim`), les mêmes dans toutes
 * les palettes et dans les deux thèmes ; le NOYAU garde la couleur d'équipe.
 *
 * ÉPAISSEURS NOMMÉES. La bande garde l'épaisseur historique du liseré (`OUTLINE_PAD`), le filet
 * s'y ajoute vers l'extérieur : le pion grandit de `MARKER_RIM_PX` de chaque côté, pas plus. Pour un
 * losange, les deux épaisseurs se prennent sur le rayon circonscrit, comme avant (la forme change,
 * pas la mesure).
 *
 * LES ANNEAUX D'ÉTAGE ET LE DOUBLE CONTOUR DU JOUEUR NE SE POSENT PLUS SUR LE LISERÉ D'AVANT :
 * ils se posent au-delà du bord extérieur du cerne (`MARKER_OUTLINE_PX`), sans quoi le premier
 * anneau d'un pion haut (rayon fixe 6,5 px) tomberait sous le filet blanc dès le deuxième étage et
 * se lirait comme lui.
 */
import { CORE_RADIUS, OUTLINE_PAD } from '@/lib/replay/screenSizes'

import { FLOOR_RING_WIDTH, floorRingRadius } from './floorRings'

/** Accroissement du rayon du noyau par étage au-dessus du sol (pixels d'écran). */
export const CORE_PER_FLOOR = 0.7

/** Épaisseur de la bande sombre, contre le noyau (pixels d'écran) : celle du liseré historique. */
export const MARKER_BAND_PX = OUTLINE_PAD
/** Épaisseur du filet clair, au bord extérieur de la bande (pixels d'écran). */
export const MARKER_RIM_PX = 0.7
/** Épaisseur totale du cerne, du bord du noyau au bord extérieur du filet. */
export const MARKER_OUTLINE_PX = MARKER_BAND_PX + MARKER_RIM_PX

/**
 * Recouvrement du filet sous la bande (pixels d'écran) : sans lui, le bord de la bande et le bord
 * intérieur du filet, tous deux anticrénelés au même endroit, laissent filtrer le sol en un fin
 * trait clair ou sombre.
 */
export const MARKER_RIM_OVERLAP_PX = 0.25

/**
 * Distance, du bord extérieur du cerne à la LIGNE MÉDIANE du premier anneau d'étage : l'anneau (1 px)
 * laisse ainsi un jour de 0,7 px entre son bord intérieur et le filet.
 */
export const FLOOR_RING_CLEARANCE_PX = 1.2

/** Les deux encres du cerne, résolues par l'appelant depuis les jetons (cf. `useReplayInks`). */
export interface MarkerOutlineInk {
  /** Bande sombre, contre le noyau (`marker-outline-band`). */
  band: string
  /** Filet clair, au bord extérieur (`marker-outline-rim`). */
  rim: string
}

/** markerCorePx : rayon du noyau (pixels d'écran, avant le facteur de densité) à l'étage `fl`. */
export function markerCorePx(fl: number): number {
  return CORE_RADIUS + CORE_PER_FLOOR * fl
}

/** markerOutlineRadius : rayon du bord extérieur du cerne (pixels d'écran, avant le facteur de densité). */
export function markerOutlineRadius(fl: number): number {
  return markerCorePx(fl) + MARKER_OUTLINE_PX
}

/**
 * firstFloorRingRadius : rayon du PREMIER anneau d'étage d'un pion à l'étage `fl` — détaché du cerne
 * de `FLOOR_RING_CLEARANCE_PX`, et suivi des suivants au pas commun (`floorRingRadius`).
 */
export function firstFloorRingRadius(fl: number): number {
  return markerOutlineRadius(fl) + FLOOR_RING_CLEARANCE_PX
}

/**
 * markerOuterEdge : rayon du bord extérieur de ce que le pion dessine AVANT l'identité du joueur de
 * la page — les anneaux d'étage s'il y en a (le dernier), le cerne sinon. C'est de là que partent le
 * double contour du joueur de la page (au-delà des anneaux d'étage, jamais mêlé à eux) et le nom écrit
 * dessous.
 */
export function markerOuterEdge(fl: number): number {
  if (fl <= 0) return markerOutlineRadius(fl)
  return floorRingRadius(fl, firstFloorRingRadius(fl)) + FLOOR_RING_WIDTH / 2
}

/**
 * drawMarkerOutline peint le cerne : le filet clair (un anneau, peint une seule fois), puis la bande
 * sombre (un disque) par-dessus ; le noyau se peint ensuite, par l'appelant.
 *
 * `addPath(r)` AJOUTE au chemin courant le sous-chemin de la forme au rayon `r` (pixels de canevas),
 * sans `beginPath` : le filet est un anneau tracé en remplissage pair-impair, donc jamais empilé sur
 * la bande — une opacité de pion inférieure à 1 (lacune de réplication) ne le fait pas se mélanger
 * à elle. `alpha` est l'opacité du pion et `k` la densité du canevas.
 */
export function drawMarkerOutline(
  ctx: CanvasRenderingContext2D,
  addPath: (radius: number) => void,
  fl: number,
  k: number,
  ink: MarkerOutlineInk,
  alpha: number,
): void {
  const core = markerCorePx(fl)
  ctx.globalAlpha = alpha
  ctx.fillStyle = ink.rim
  ctx.beginPath()
  addPath((core + MARKER_OUTLINE_PX) * k)
  addPath((core + MARKER_BAND_PX - MARKER_RIM_OVERLAP_PX) * k)
  ctx.fill('evenodd')
  ctx.fillStyle = ink.band
  ctx.beginPath()
  addPath((core + MARKER_BAND_PX) * k)
  ctx.fill()
}
