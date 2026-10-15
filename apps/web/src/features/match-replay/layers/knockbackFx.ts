/**
 * knockbackFx.ts — LA POUSSÉE DU RÉPULSEUR : une onde qui part de l'auteur, et un sillage sur le
 * joueur poussé, orienté par l'impulsion lue.
 *
 * LA SOURCE est le document (schéma 94) : `knockbacks`, une entrée par joueur poussé — (t, slot),
 * l'auteur `by` et la vitesse imposée `vx/vy/vz`, lus dans la paire de messages que le film écrit
 * (`EquipmentObjectKnockedBack` puis `EquipmentKnockbackPlayer`). Aucun seuil, aucune lecture de
 * trajectoire pour décider : la poussée EST publiée.
 *
 * ## Ce qui se dessine
 *
 *  - L'ONDE, centrée sur l'auteur à l'image de la poussée : un anneau qui s'étend et s'efface. Elle
 *    dit QUI a poussé. Sans auteur publié (`by` absent), pas d'onde : on ne pose pas une source au
 *    hasard.
 *  - LE SILLAGE sur le joueur poussé, le tracé du propulseur (`drawDash` de `thrusterDashFx.ts`) :
 *    la DIRECTION vient de l'impulsion (`vx`, `vy`), pas du déplacement. Sans impulsion publiée,
 *    pas de sillage.
 *
 * LA COULEUR est celle de l'équipe de l'AUTEUR, résolue à l'image de la poussée (le slot d'une vie
 * est repris aux réapparitions) ; sans auteur, celle du joueur poussé.
 *
 * Tout est en géométrie pure sur le contexte ; les encres arrivent déjà résolues depuis les tokens.
 */
import type { ReplayPoint } from '@/lib/api/types'

import { positionAt, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { buildLivesBySlot, lifeOfSlotAt } from '../model/livesPosition'
import { type CanvasView, projectTo } from '../model/replayView'
import { dashProgress, drawDash } from './thrusterDashFx'

/** DURÉE DE L'EFFET, en temps réel du match : un souffle bref, comme le dash du propulseur. */
export const KNOCKBACK_FX_MS = 520

/** Rayon final de l'onde, en pixels d'écran (multiplié par la densité). */
const WAVE_RADIUS = 26
const WAVE_WIDTH = 2.2
const WAVE_ALPHA = 0.85
/** Sous mouvement réduit : l'anneau est posé, plein rayon, opacité constante. */
const WAVE_ALPHA_STATIC = 0.5

/** Une poussée prête à dessiner. */
export interface KnockbackFx {
  frame: number
  /** Slot du joueur poussé, et les points de SA vie (le sillage suit le pion). */
  slot: number
  points: ReplayPoint[]
  /** Slot de l'auteur et sa position à l'image de la poussée ; absents sans auteur publié. */
  by: number | null
  origin: XY | null
  /** Vitesse imposée, plan horizontal, en monde ; absente sans impulsion publiée. */
  push: XY | null
}

/**
 * buildKnockbackFx joint chaque poussée à LA VIE du joueur poussé qui couvre son instant et, quand
 * l'auteur est publié, à la position de la vie de l'auteur à cet instant. Une poussée dont la vie
 * poussée n'a pas de points ne produit rien.
 */
export function buildKnockbackFx(doc: ReplayDocumentReady): KnockbackFx[] {
  if (doc.knockbacks.length === 0) return []
  const bySlot = buildLivesBySlot(doc.tracks)
  const out: KnockbackFx[] = []
  for (const k of doc.knockbacks) {
    const pushed = lifeOfSlotAt(bySlot, k.slot, k.t)
    if (!pushed || pushed.points.length === 0) continue
    let origin: XY | null = null
    if (k.by != null) {
      const author = lifeOfSlotAt(bySlot, k.by, k.t)
      origin = author ? positionAt(author.points, k.t) : null
    }
    const push = k.vx != null && k.vy != null ? { x: k.vx, y: k.vy } : null
    out.push({
      frame: k.t,
      slot: k.slot,
      points: pushed.points,
      by: k.by ?? null,
      origin,
      push,
    })
  }
  return out
}

/** L'horloge du calque : image courante, durée réelle d'une image, densité, préférence. */
interface KnockbackTime {
  frame: number
  frameMs: number
  k: number
  reducedMotion: boolean
}

/** Le style : la couleur d'équipe d'une vie, à une image donnée. */
export interface KnockbackStyle {
  colorOfSlot: (slot: number, frame: number) => string | null
}

/**
 * waveProgress : rayon (part du rayon final) et présence de l'onde à cet âge, ou `null` hors de la
 * fenêtre. Étirement easeOutCubic, comme le dash et l'onde de choc de l'explosion.
 */
export function waveProgress(
  ageMs: number,
  reducedMotion: boolean,
): { reach: number; alpha: number } | null {
  if (!(ageMs >= 0) || ageMs > KNOCKBACK_FX_MS) return null
  if (reducedMotion) return { reach: 1, alpha: WAVE_ALPHA_STATIC }
  const p = ageMs / KNOCKBACK_FX_MS
  return { reach: 1 - Math.pow(1 - p, 3), alpha: (1 - p) * WAVE_ALPHA }
}

/** drawKnockbackLayer peint les poussées actives à l'image courante. */
export function drawKnockbackLayer(
  ctx: CanvasRenderingContext2D,
  fx: readonly KnockbackFx[],
  view: CanvasView,
  time: KnockbackTime,
  style: KnockbackStyle,
): void {
  for (const f of fx) {
    const ageMs = (time.frame - f.frame) * time.frameMs
    const wave = waveProgress(ageMs, time.reducedMotion)
    if (!wave) continue
    const color =
      (f.by != null ? style.colorOfSlot(f.by, f.frame) : null) ?? style.colorOfSlot(f.slot, f.frame)
    if (!color) continue
    ctx.save()
    if (f.origin) drawWave(ctx, projectTo(view, f.origin), wave, { k: time.k, color })
    const dash = dashProgress(ageMs, time.reducedMotion)
    const head = positionAt(f.points, time.frame)
    const dir = f.push && head ? screenDirection(head, f.push, view) : null
    if (dash && head && dir) drawDash(ctx, projectTo(view, head), dir, dash, { k: time.k, color })
    ctx.restore()
  }
}

function drawWave(
  ctx: CanvasRenderingContext2D,
  c: XY,
  wave: { reach: number; alpha: number },
  paint: { k: number; color: string },
): void {
  const r = WAVE_RADIUS * paint.k * wave.reach
  if (!(r > 0)) return
  ctx.globalAlpha = wave.alpha
  ctx.strokeStyle = paint.color
  ctx.lineWidth = WAVE_WIDTH * paint.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.stroke()
}

/**
 * screenDirection projette le pion et le pion décalé de l'impulsion, et normalise EN PIXELS (la
 * projection peut inverser un axe et n'a pas la même échelle en x et en y).
 */
function screenDirection(at: XY, push: XY, view: CanvasView): XY | null {
  const a = projectTo(view, at)
  const b = projectTo(view, { x: at.x + push.x, y: at.y + push.y })
  const dx = b.x - a.x
  const dy = b.y - a.y
  const len = Math.hypot(dx, dy)
  if (!(len > 0)) return null
  return { x: dx / len, y: dy / len }
}
