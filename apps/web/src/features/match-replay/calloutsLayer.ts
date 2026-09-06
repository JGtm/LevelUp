/**
 * calloutsLayer.ts — les ZONES NOMMÉES (« callouts ») de la carte : normalisation,
 * affectation 3D (zoneAt) et calque canvas. Portage du POC (artefact eb7b8af2,
 * l.1033-1094 et l.780-789), qui est LA spec de rendu — pas une réinvention.
 *
 * CE QUE LE POC A ÉTABLI, et qui se retrouve ici tel quel :
 * - les GRANDES zones PAVENT la carte (mesuré sur Ridgeline : 100 % des cellules
 *   couvertes par exactement une zone) : aplat léger en règle PAIR-IMPAIR (elles ont des
 *   trous et parfois plusieurs parties depuis le découpage) + frontière ;
 * - les zones FINES sont des étages imbriqués qui se superposent en 2D : on ne les
 *   remplit pas (la carte deviendrait illisible), teinte NEUTRE unique, contour
 *   pointillé — l'empilement se lit « niveau inférieur », pas « zones concurrentes » ;
 * - le libellé s'écrit UNE fois par nom, sur la zone la plus haute, en MAJUSCULES,
 *   BLANC CERNÉ DE NOIR — le cerne le rend lisible sur les aplats clairs comme sur le
 *   fond sombre, SANS dépendre du thème ;
 * - l'affectation d'un joueur à sa zone se fait en distance 3D : une zone existe en
 *   version haute ET basse (« Fer à cheval » / « Fer à cheval inférieur ») et les étages
 *   se confondent en 2D.
 */
import type { ReplayCalloutZone, ReplayMapCallouts } from '@/lib/api/types'

import type { ReplayLocale } from './i18n/i18n'
import { type XY } from './replayLogic'
import { type CanvasView, projectTo } from './replayView'

/** Une zone prête à dessiner : nullabilité du transport résolue, ancre de libellé posée. */
export interface CalloutZoneReady {
  fr: string
  en: string
  /** Centre 3D du volume (mètres monde) — la référence de zoneAt. */
  x: number
  y: number
  z: number
  big: boolean
  /** Contour principal ([] = zone non dessinée, interrogeable par zoneAt seulement). */
  polygon: XY[]
  /** Parties supplémentaires et trous (provenance « découpe ») — remplissage pair-impair. */
  parts: XY[][]
  holes: XY[][]
  /** Ancre du libellé : centroïde du contour (le POC écrivait au centroïde précalculé). */
  labelX: number
  labelY: number
}

/**
 * normalizeCallouts résout la nullabilité du transport UNE fois, à l'entrée (même règle
 * que replayNormalize). TOUTES les zones sont gardées, y compris sans polygone : les
 * volumes secondaires d'un même nom portent les étages bas et rendent zoneAt juste —
 * seul le DESSIN exige un contour.
 */
export function normalizeCallouts(entry: ReplayMapCallouts | null | undefined): CalloutZoneReady[] {
  if (!entry?.zones) return []
  return entry.zones.map((z) => {
    const polygon = ring(z.polygon)
    const anchor = polygon.length >= 3 ? centroid(polygon) : { x: z.x, y: z.y }
    return {
      fr: z.fr,
      en: z.en,
      x: z.x,
      y: z.y,
      z: z.z,
      big: z.big ?? false,
      polygon,
      parts: rings(z.parts),
      holes: rings(z.holes),
      labelX: anchor.x,
      labelY: anchor.y,
    }
  })
}

function ring(pts: ReplayCalloutZone['polygon']): XY[] {
  if (!pts) return []
  const out: XY[] = []
  for (const p of pts) {
    if (p && p.length >= 2) out.push({ x: p[0], y: p[1] })
  }
  return out
}

function rings(list: ReplayCalloutZone['parts']): XY[][] {
  if (!list) return []
  const out: XY[][] = []
  for (const r of list) {
    const done = ring(r ?? null)
    if (done.length >= 3) out.push(done)
  }
  return out
}

function centroid(pts: XY[]): XY {
  let sx = 0
  let sy = 0
  for (const p of pts) {
    sx += p.x
    sy += p.y
  }
  return { x: sx / pts.length, y: sy / pts.length }
}

/**
 * zoneAt rend la zone dont le CENTRE 3D est le plus proche du point — le portage exact
 * du POC. La distance est euclidienne 3D : c'est le z qui départage les étages
 * superposés, un plus-proche-voisin 2D rendrait l'étage du dessus.
 */
export function zoneAt(zones: CalloutZoneReady[], x: number, y: number, z: number): CalloutZoneReady | null {
  let best: CalloutZoneReady | null = null
  let bd = Infinity
  for (const c of zones) {
    const dx = c.x - x
    const dy = c.y - y
    const dz = c.z - z
    const d = dx * dx + dy * dy + dz * dz
    if (d < bd) {
      bd = d
      best = c
    }
  }
  return best
}

/** calloutLabel rend le libellé officiel dans la langue de l'interface. */
export function calloutLabel(zone: CalloutZoneReady, locale: ReplayLocale): string {
  return locale === 'fr' ? zone.fr : zone.en
}

/** Cadrage du canvas (les mêmes paramètres que worldToCanvas, forme de replayDraw). */
/** Style du calque : couleurs déjà RÉSOLUES par l'appelant (règle color-tokens). */
export interface CalloutsStyle {
  /** Une couleur par grande zone (série cyclée) — la rotation de teinte du POC. */
  bigColors: string[]
  /** Encre neutre unique des zones fines (elles ne se concurrencent pas). */
  fineInk: string
  locale: ReplayLocale
}

// Réglages du POC, repris tels quels (ils y ont été réglés à l'écran).
const FINE_FILL_ALPHA = 0.05
const FINE_STROKE_ALPHA = 0.22
const FINE_DASH = [3, 2]
const BIG_FILL_ALPHA = 0.055
const BIG_STROKE_ALPHA = 0.28
/**
 * Libellé : 9,5 px d'ÉCRAN.
 *
 * IL ÉTAIT À 25 px, ET C'ÉTAIT UNE ERREUR D'UNITÉ. Le POC affichait « 25 » parce que son
 * canevas de 1 600 px était rendu à ~840 px : ses 25 px de dessin valaient ~13 px à l'écran.
 * ICI le contexte est déjà transformé au devicePixelRatio et le canvas s'affiche à sa taille
 * CSS — une unité de dessin EST un pixel d'écran, et 25 donnait donc un libellé DEUX FOIS
 * trop grand, qui débordait de sa zone et couvrait les joueurs (planche du 16/08 : « police
 * trop grande, limiter le débordement au maximum »).
 *
 * LA BORNE EST LE NOM DE JOUEUR, PAS UN GOÛT : un nom de zone ne doit jamais peser plus lourd
 * qu'un joueur sur la carte — au plus sa taille + 1 px d'écran (8,7 + 1 = 9,7). 9,5 s'y tient.
 * Le cerne suit la même proportion qu'avant (1/5 de la police), sans quoi un contour de 5 px
 * autour d'un texte de 9,5 px mangerait les lettres.
 */
const LABEL_FONT = '600 9.5px ui-sans-serif, system-ui, sans-serif'
const LABEL_OUTLINE_WIDTH = 1.9

/**
 * drawCalloutsLayer peint les zones nommées : fines d'abord (dessous), grandes ensuite,
 * libellés en dernier. Calque STATIQUE — l'appelant le cuit hors écran et le recopie,
 * comme le sol.
 */
export function drawCalloutsLayer(
  ctx: CanvasRenderingContext2D,
  zones: CalloutZoneReady[],
  view: CanvasView,
  style: CalloutsStyle,
): void {
  const px = (p: XY) => projectTo(view, p)

  // Les zones FINES, sous les grandes : contour pointillé, teinte neutre, aplat quasi nul.
  for (const c of zones) {
    if (c.big || c.polygon.length < 3) continue
    ctx.beginPath()
    tracePoly(ctx, c.polygon, px)
    ctx.globalAlpha = FINE_FILL_ALPHA
    ctx.fillStyle = style.fineInk
    ctx.fill()
    ctx.globalAlpha = FINE_STROKE_ALPHA
    ctx.strokeStyle = style.fineInk
    ctx.lineWidth = 1
    ctx.setLineDash(FINE_DASH)
    ctx.stroke()
    ctx.setLineDash([])
  }

  // Les GRANDES zones : remplissage pair-impair (parties + trous du découpage), une
  // couleur de série par zone — la rotation de teinte du POC, en tokens résolus.
  let bigIndex = 0
  for (const c of zones) {
    if (!c.big || c.polygon.length < 3) continue
    const color = style.bigColors[bigIndex % Math.max(style.bigColors.length, 1)] ?? style.fineInk
    bigIndex++
    ctx.beginPath()
    tracePoly(ctx, c.polygon, px)
    for (const q of c.parts) tracePoly(ctx, q, px)
    for (const h of c.holes) tracePoly(ctx, [...h].reverse(), px)
    ctx.globalAlpha = BIG_FILL_ALPHA
    ctx.fillStyle = color
    ctx.fill('evenodd')
    ctx.globalAlpha = BIG_STROKE_ALPHA
    ctx.strokeStyle = color
    ctx.lineWidth = 1
    ctx.stroke()
  }
  ctx.globalAlpha = 1

  drawLabels(ctx, zones, px, style)
}

function tracePoly(ctx: CanvasRenderingContext2D, pts: XY[], px: (p: XY) => XY): void {
  pts.forEach((p, k) => {
    const c = px(p)
    if (k === 0) ctx.moveTo(c.x, c.y)
    else ctx.lineTo(c.x, c.y)
  })
  ctx.closePath()
}

/**
 * drawLabels écrit chaque nom UNE fois, sur la zone dessinée la plus haute qui le porte
 * (tri z décroissant, règle du POC — sans dédoublonnage, « Tuyaux » s'écrirait quatre
 * fois). Seules les zones AVEC contour reçoivent un libellé : les volumes secondaires
 * servent zoneAt, pas l'affichage.
 */
function drawLabels(
  ctx: CanvasRenderingContext2D,
  zones: CalloutZoneReady[],
  px: (p: XY) => XY,
  style: CalloutsStyle,
): void {
  const written = new Set<string>()
  ctx.font = LABEL_FONT
  ctx.textAlign = 'center'
  // Cerne arrondi : sans lineJoin round + miterLimit bas, les angles des lettres
  // produisent des pointes là où deux segments du contour se rejoignent (POC).
  ctx.lineJoin = 'round'
  ctx.miterLimit = 2
  ctx.lineWidth = LABEL_OUTLINE_WIDTH
  // BLANC CERNÉ DE NOIR, volontairement HORS thème (spec POC portée par le plan) : le
  // libellé se pose sur les aplats colorés de la carte, pas sur le fond de la page — le
  // cerne porte le contraste dans les deux thèmes, comme le HUD du jeu. C'est une encre
  // STRUCTURELLE de calque (exception documentée du même ordre que canvasInk.ts), pas
  // une couleur qui dit quelque chose : rien ici ne distingue un rôle métier.
  ctx.strokeStyle = 'rgba(0, 0, 0, 0.92)'
  ctx.fillStyle = 'rgba(255, 255, 255, 1)'
  const drawn = zones.filter((c) => c.polygon.length >= 3)
  drawn.sort((p, q) => q.z - p.z)
  for (const c of drawn) {
    const label = calloutLabel(c, style.locale)
    if (!label || written.has(label)) continue
    written.add(label)
    const at = px({ x: c.labelX, y: c.labelY })
    const text = label.toUpperCase()
    ctx.globalAlpha = c.big ? 1 : 0.9
    ctx.strokeText(text, at.x, at.y)
    ctx.fillText(text, at.x, at.y)
  }
  ctx.globalAlpha = 1
  ctx.textAlign = 'left'
}
