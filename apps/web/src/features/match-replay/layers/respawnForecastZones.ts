/**
 * respawnForecastZones.ts — LA TEINTE DES ZONES de la probabilité de réapparition : chaque zone de
 * l'échantillon courant d'un joueur mort se teinte, en fond léger, à la couleur de son pion, d'autant
 * plus que sa probabilité approche celle de la première zone. Au point réel, la teinte se retient de
 * moitié et la zone qui nomme le point est cernée.
 *
 * LE CALQUE SE POSE SOUS LES PIONS, avec le vocabulaire du terrain (`replayCompose`) : une teinte n'est
 * pas un événement. Les repères (anneau, étiquettes, point réel, légende) sont un autre calque, au-dessus.
 *
 * LE ROGNAGE AU PLAN EST OPTIONNEL (`clip`). Une zone nommée peut déborder du plan de la carte : avec
 * un rognage, la teinte se peint sur une toile de travail, le masque en retire ce qui sort du plan,
 * puis la toile se recopie sur la scène. Sans rognage, la teinte se peint directement.
 */
import type { CanvasView } from '../model/replayView'
import type { ForecastMoment } from '../model/respawnForecast.logic'
import { tintAlpha } from '../model/respawnForecast.logic'
import { hasShape, traceZone, type ForecastPaint } from './respawnForecastPaint'

/** Au point réel, la teinte se retient de moitié : le regard va au point. */
const REVEAL_TINT_FACTOR = 0.5
const OUTLINE_WIDTH_PX = 3

/** Une toile de travail, avec son contexte, à la densité de la scène. */
export interface ForecastScratch {
  canvas: CanvasImageSource
  ctx: CanvasRenderingContext2D
}

/** Le rognage de la teinte au plan de la carte. */
export interface ZoneClip {
  /** Une toile de travail VIDE de la taille demandée (pixels de dessin) ; `null` = indisponible, pas de rognage. */
  scratch: (width: number, height: number) => ForecastScratch | null
  /** Retire de `ctx` ce qui sort du plan (composition `destination-in` de l'image plan). */
  mask: (ctx: CanvasRenderingContext2D, view: CanvasView) => void
}

/**
 * createForecastScratch : la toile de travail du rognage, créée au premier besoin et retaillée seulement si
 * la scène change de taille (retailler efface et réalloue). Vidée à chaque demande, repère remis à zéro.
 */
export function createForecastScratch(): ZoneClip['scratch'] {
  let canvas: HTMLCanvasElement | null = null
  return (width, height) => {
    if (typeof document === 'undefined') return null
    canvas ??= document.createElement('canvas')
    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width
      canvas.height = height
    }
    const ctx = canvas.getContext('2d')
    if (!ctx) return null
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.clearRect(0, 0, width, height)
    return { canvas, ctx }
  }
}

/** Les entrées de la teinte : celles de tous les peintres, et le rognage s'il y en a un. */
export interface ZonesPaint extends ForecastPaint {
  clip?: ZoneClip
}

function paintTints(ctx: CanvasRenderingContext2D, moments: readonly ForecastMoment[], view: CanvasView, p: ZonesPaint): void {
  for (const m of moments) {
    const color = p.colorOf(m.death.xuid)
    const factor = (m.phase === 'reveal' ? REVEAL_TINT_FACTOR : 1) * m.opacity
    ctx.fillStyle = color
    for (const z of m.sample.zones) {
      const zone = p.zones.get(z.zone)
      if (!hasShape(zone?.shape)) continue
      traceZone(ctx, zone.shape, view)
      ctx.globalAlpha = tintAlpha(z.p, m.peak) * factor
      ctx.fill('evenodd')
    }
    if (m.phase !== 'reveal' || m.death.actualZone === null) continue
    const real = p.zones.get(m.death.actualZone)
    if (!hasShape(real?.shape)) continue
    traceZone(ctx, real.shape, view)
    ctx.globalAlpha = m.opacity
    ctx.strokeStyle = color
    ctx.lineWidth = OUTLINE_WIDTH_PX * p.k
    ctx.stroke()
  }
  ctx.globalAlpha = 1
}

/**
 * drawForecastZones teinte les zones des morts actives. Sans mort active, rien n'est émis.
 */
export function drawForecastZones(
  ctx: CanvasRenderingContext2D,
  moments: readonly ForecastMoment[],
  view: CanvasView,
  p: ZonesPaint,
): void {
  if (moments.length === 0) return
  const work = p.clip?.scratch(Math.ceil(view.width * p.k), Math.ceil(view.height * p.k)) ?? null
  if (!work || !p.clip) {
    paintTints(ctx, moments, view, p)
    return
  }
  work.ctx.save()
  work.ctx.setTransform(p.k, 0, 0, p.k, 0, 0)
  paintTints(work.ctx, moments, view, p)
  p.clip.mask(work.ctx, view)
  work.ctx.restore()
  ctx.drawImage(work.canvas, 0, 0, view.width, view.height)
}
