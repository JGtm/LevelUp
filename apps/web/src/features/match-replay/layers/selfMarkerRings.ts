/**
 * selfMarkerRings.ts — L'IDENTITÉ DU JOUEUR DE LA PAGE SUR LA CARTE : le double contour et le halo
 * que porte son pion (forme 'ring'), posés au-delà de tout ce que le pion dessine d'autre.
 *
 * Extrait de `replayMarkers.ts` : ce bloc ne dépend que de la couleur `selfInk`, de la densité du
 * canevas et de la géométrie du pion (`markerOutline.ts`).
 */
import type { XY } from '../../../lib/replay/replayLogic'
import { markerOuterEdge } from './markerOutline'

/** Ce que ces tracés lisent du style du calque. */
export interface SelfRingsStyle {
  /** Encre du double contour et du halo (token `success`). */
  selfInk: string
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
}

/**
 * LE JOUEUR DE LA PAGE (forme 'ring') — DOUBLE CONTOUR ET HALO depuis le 2026-08-18.
 *
 * CE QUE LE RETOUR DEMANDE : « avoir l'icône du joueur actif qui se démarque de tous les
 * autres aussi (j'aurais bien aimé du vert mais pour l'accessibilité je sais pas si ça peut
 * le faire) ». La réponse tient en une règle : LA COULEUR NE PORTE JAMAIS SEULE. Le marqueur
 * du joueur de la page était déjà cerclé d'UN anneau ; il en porte DEUX, plus un halo diffus,
 * et c'est cette FORME qui le distingue — un lecteur qui ne voit pas la teinte voit toujours
 * le seul point de la carte à deux anneaux. La couleur (`selfInk`, token `success`) vient EN
 * PLUS, sur le contour et le halo ; le NOYAU garde la couleur d'ÉQUIPE, qui dit le camp.
 *
 * LE HALO EST DE RETOUR ICI, ET SEULEMENT ICI. Il avait été supprimé de TOUS les marqueurs le
 * 16/08 (« la lueur diffuse doublait l'emprise et empâtait la carte ») — sur un seul point de
 * la carte, il ne l'empâte pas, il le désigne.
 */
const SELF_RING_WIDTH = 1.5
const SELF_RING_GAP = 1.6
/** Écart entre les deux anneaux d'identité, en pixels d'écran. */
const SELF_RING_GAP_2 = 2.4
/** Le halo : un disque diffus posé sous le marqueur, au rayon du second anneau. */
const SELF_HALO_PAD = 2.2
const SELF_HALO_ALPHA = 0.22

/**
 * selfRingRadius : rayon du PREMIER anneau d'identité du joueur de la page (forme 'ring'), posé au-delà
 * de tout ce que le pion dessine d'autre (cerne, anneaux d'étage) : les deux familles d'anneaux ne
 * se superposent jamais, à quelque étage que ce soit.
 */
function selfRingRadius(fl: number): number {
  return markerOuterEdge(fl) + SELF_RING_GAP
}

/** selfRingsEdge : rayon du bord extérieur du halo, c'est-à-dire de tout ce que l'identité ajoute au pion. */
export function selfRingsEdge(fl: number): number {
  return selfRingRadius2(fl) + SELF_HALO_PAD
}

/** selfRingRadius2 : rayon du SECOND anneau — c'est lui qui fait le « double contour ». */
function selfRingRadius2(fl: number): number {
  return selfRingRadius(fl) + SELF_RING_GAP_2
}

/**
 * drawSelfRings — LES DEUX ANNEAUX du joueur de la page, à `selfInk`.
 *
 * DEUX, PAS UN : c'est le double contour qui porte l'identité quand la couleur ne peut pas
 * (cf. la note de SELF_RING_WIDTH). Le second est plus fin et plus pâle — l'œil lit « un
 * anneau souligné », pas « deux cercles ».
 */
export function drawSelfRings(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: SelfRingsStyle,
  fl: number,
  alpha: number,
): void {
  ctx.strokeStyle = style.selfInk
  ctx.globalAlpha = alpha
  ctx.lineWidth = SELF_RING_WIDTH * style.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, selfRingRadius(fl) * style.k, 0, Math.PI * 2)
  ctx.stroke()
  ctx.globalAlpha = 0.75 * alpha
  ctx.lineWidth = (SELF_RING_WIDTH / 2) * style.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, selfRingRadius2(fl) * style.k, 0, Math.PI * 2)
  ctx.stroke()
  ctx.globalAlpha = 1
}

/** drawSelfHalo — la lueur diffuse sous le marqueur du joueur de la page (dégradé radial). */
export function drawSelfHalo(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: SelfRingsStyle,
  fl: number,
  alpha: number,
): void {
  const r = (selfRingRadius2(fl) + SELF_HALO_PAD) * style.k
  const gradient = ctx.createRadialGradient(c.x, c.y, 0, c.x, c.y, r)
  gradient.addColorStop(0, style.selfInk)
  gradient.addColorStop(1, 'transparent')
  ctx.globalAlpha = SELF_HALO_ALPHA * alpha
  ctx.fillStyle = gradient
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = 1
}
