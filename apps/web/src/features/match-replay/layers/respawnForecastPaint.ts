/**
 * respawnForecastPaint.ts — CE QUE PARTAGENT LES PEINTRES DE LA PROBABILITÉ DE RÉAPPARITION : les
 * types de leurs entrées (encres, textes, zones), la pastille d'étiquette et le tracé d'une zone.
 *
 * AUCUN TEXTE ET AUCUNE COULEUR ÉCRITS ICI : les textes arrivent résolus dans la langue de la page
 * (`ForecastWording`, bâti par `useReplayRespawnForecast`), les encres des jetons (`ForecastInk`,
 * `useReplayInks`) et la couleur d'un joueur est celle de son pion (`colorOf`). Les tailles sont en
 * pixels d'écran multipliées par la densité `k`, comme celles des autres calques.
 */
import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import { zoneHasShape } from '@/lib/replay/calloutShape'
import { roundedRectPath } from '@/lib/replay/roundedRect'

import type { XY } from '../../../lib/replay/replayLogic'
import { projectTo, type CanvasView } from '../model/replayView'
import type { ForecastDraw } from '../model/respawnForecast.logic'

/** Une zone nommée de la probabilité : son nom dans la langue de la page et sa forme (`volume_index`). */
export interface ForecastZone {
  name: string
  shape: CalloutZoneReady
}

/** Les quatre encres du calque : la pastille et son texte (voile et texte sur voile), le liseré, le repli. */
export interface ForecastInk {
  panel: string
  onPanel: string
  underlay: string
  neutral: string
}

/** Tous les textes que le calque écrit sur la carte, déjà résolus dans la langue de la page. */
export interface ForecastWording {
  title: string
  chip: (name: string, seconds: number) => string
  rank: (rank: number) => string
  /** Une probabilité, `0.35` -> « 35 % ». */
  share: (p: number) => string
  /** « Disque : probabilité du point · étiquette : somme de la zone ». */
  legendPoints: string
  /** « Teinte légère : probabilité de la zone » (sous les points). */
  legendTintUnderPoints: string
  legendCountdown: (name: string, seconds: number) => string
  appearedIn: (name: string, zone: string) => string
  appearedOutside: (name: string) => string
  appearedRank: (rank: string, share: string) => string
  appearedUnlikely: string
  legendScale: string
  legendFooter: string
}

/** Ce que tous les peintres reçoivent, en plus des morts actives et de la vue. */
export interface ForecastPaint {
  /** Densité du canevas. */
  k: number
  /** Ce que le calque dessine : zones et points, zones seules, points seuls. */
  draw: ForecastDraw
  ink: ForecastInk
  wording: ForecastWording
  /** Les zones du document par `volume_index`, nom et forme. */
  zones: ReadonlyMap<number, ForecastZone>
  /** La couleur du pion du joueur mort (équipe, ou série distincte) ; le repli neutre sinon. */
  colorOf: (xuid: string) => string
  nameOf: (xuid: string) => string
}

const FONT_FAMILY = 'ui-sans-serif, system-ui, sans-serif'
const PILL_PAD_X_PX = 6
const PILL_PAD_Y_PX = 3
const PILL_RADIUS_PX = 5
const PILL_BORDER_PX = 1.2
const PILL_ALPHA = 0.78

/** La police d'un texte du calque, à la densité du canevas. */
export function forecastFont(px: number, weight: number, k: number): string {
  return `${weight} ${px * k}px ${FONT_FAMILY}`
}

/** Le tracé d'une zone : contour, parties supplémentaires et trous (remplissage pair-impair). */
export function traceZone(ctx: CanvasRenderingContext2D, zone: CalloutZoneReady, view: CanvasView): void {
  ctx.beginPath()
  const ring = (pts: readonly XY[]) => {
    pts.forEach((p, i) => {
      const c = projectTo(view, p)
      if (i === 0) ctx.moveTo(c.x, c.y)
      else ctx.lineTo(c.x, c.y)
    })
    ctx.closePath()
  }
  ring(zone.polygon)
  for (const part of zone.parts) ring(part)
  for (const hole of zone.holes) ring([...hole].reverse())
}

/** La zone se dessine-t-elle : elle a une forme (`zoneHasShape`). */
export function hasShape(zone: CalloutZoneReady | undefined): zone is CalloutZoneReady {
  return !!zone && zoneHasShape(zone)
}

/** Où s'ancre une pastille par rapport à son point. */
export type PillAnchor = 'above' | 'below' | 'center'

export interface PillStyle {
  k: number
  ink: ForecastInk
  /** Couleur du liseré (celle du joueur). */
  border: string
  alpha: number
  anchor: PillAnchor
  /** Taille du texte, en pixels d'écran. */
  fontPx: number
  /** Décalage vertical entre le point et la pastille, en pixels d'écran. */
  gapPx: number
}

/**
 * drawPill pose une pastille de texte (une ou deux lignes) centrée sur `at` : voile sombre, liseré à la
 * couleur du joueur, texte sur voile. Rend la boîte posée, pour qu'un appelant s'y accroche.
 */
export function drawPill(
  ctx: CanvasRenderingContext2D,
  at: XY,
  lines: readonly string[],
  s: PillStyle,
): { x: number; y: number; w: number; h: number } {
  const { k } = s
  ctx.save()
  ctx.font = forecastFont(s.fontPx, 600, k)
  const lineH = s.fontPx * 1.25 * k
  const textW = Math.max(...lines.map((l) => ctx.measureText(l).width))
  const w = textW + PILL_PAD_X_PX * 2 * k
  const h = lineH * lines.length + PILL_PAD_Y_PX * 2 * k
  const x = at.x - w / 2
  const y = s.anchor === 'above' ? at.y - s.gapPx * k - h : s.anchor === 'below' ? at.y + s.gapPx * k : at.y - h / 2
  ctx.globalAlpha = PILL_ALPHA * s.alpha
  ctx.fillStyle = s.ink.panel
  roundedRectPath(ctx, { x, y, w, h }, PILL_RADIUS_PX * k)
  ctx.fill()
  ctx.globalAlpha = s.alpha
  ctx.strokeStyle = s.border
  ctx.lineWidth = PILL_BORDER_PX * k
  ctx.stroke()
  ctx.fillStyle = s.ink.onPanel
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  lines.forEach((line, i) => ctx.fillText(line, at.x, y + PILL_PAD_Y_PX * k + lineH * (i + 0.5)))
  ctx.restore()
  return { x, y, w, h }
}
