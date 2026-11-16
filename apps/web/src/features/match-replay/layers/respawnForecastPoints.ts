/**
 * respawnForecastPoints.ts — LES POINTS PESÉS de la probabilité de réapparition : chaque point candidat
 * de la carte devient un disque creux à la couleur du joueur mort, dont le rayon suit la racine de sa
 * probabilité à l'échantillon courant (`discRadiusPx`). Posés par-dessus la teinte des zones, sous les
 * pions, dans le même calque que la teinte.
 *
 * AU POINT RÉEL, les disques s'estompent (le regard va au point réel, que les équerres du calque de
 * repères verrouillent) ; ils s'effacent avec le reste. Un point sous `DISC_MIN_P`, ou dont la position
 * n'est pas finie, ne se dessine pas ; rien n'est deviné pour un échantillon dont les probabilités de
 * points ne sont pas alignées (liste vide : aucun disque).
 */
import { projectTo, type CanvasView } from '../model/replayView'
import { DISC_MIN_P, discRadiusPx, type ForecastMoment } from '../model/respawnForecast.logic'
import type { ForecastPaint } from './respawnForecastPaint'

const DISC_FILL_ALPHA = 0.32
const DISC_STROKE_WIDTH_PX = 1.5
/** Au point réel, l'opacité des disques. */
const REVEAL_DISC_FACTOR = 0.18

/**
 * drawForecastPoints peint les disques des morts actives. Sans mort active ou sans probabilité de point,
 * rien n'est émis.
 */
export function drawForecastPoints(
  ctx: CanvasRenderingContext2D,
  moments: readonly ForecastMoment[],
  view: CanvasView,
  p: ForecastPaint,
): void {
  let drawn = false
  for (const m of moments) {
    const probabilities = m.sample.pointsP
    if (probabilities.length === 0) continue
    drawn = true
    const color = p.colorOf(m.death.xuid)
    const factor = (m.phase === 'reveal' ? REVEAL_DISC_FACTOR : 1) * m.opacity
    ctx.fillStyle = color
    ctx.strokeStyle = color
    ctx.lineWidth = DISC_STROKE_WIDTH_PX * p.k
    probabilities.forEach((pp, i) => {
      const point = m.points[i]
      if (!point || !(pp >= DISC_MIN_P) || !Number.isFinite(point.x) || !Number.isFinite(point.y)) return
      const c = projectTo(view, point)
      ctx.beginPath()
      ctx.arc(c.x, c.y, discRadiusPx(pp) * p.k, 0, Math.PI * 2)
      ctx.globalAlpha = DISC_FILL_ALPHA * factor
      ctx.fill()
      ctx.globalAlpha = factor
      ctx.stroke()
    })
  }
  if (drawn) ctx.globalAlpha = 1
}
