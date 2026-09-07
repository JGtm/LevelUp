/**
 * grenadeRestLayer.ts — LA FIN DE VOL D'UNE GRENADE : explosion, nappe électrique, halo.
 *
 * EXTRAIT DE `replayDraw.ts` LE 2026-08-18 (lot R3, item R3.2). Le fichier portait sept
 * calques et franchissait le seuil de 500 lignes du dépôt ; la nappe Dynamo devait y gagner
 * une graphie et une doctrine de durée. La découpe tombe sur une frontière nette : ce
 * fichier-ci ne connaît QUE la fin de vol d'une grenade, et rien d'autre du canvas ne la
 * connaît. Aucune règle n'a changé au passage — seule la nappe, ci-dessous, est neuve.
 *
 * CE QUE LE POINT SIGNIFIE, et le rendu doit le respecter : la DERNIÈRE POSITION RÉPLIQUÉE
 * du projectile, JAMAIS un impact — le film n'enregistre aucune détonation. L'EXPLOSION
 * POSÉE LÀ EST DONC UNE MISE EN SCÈNE ASSUMÉE (item 3.2), et c'est pourquoi l'écran continue
 * de dire « dernière position connue ».
 */
import { drawExplosion } from './explosionFx'
import type { FxInk } from './fxInk'
import { explosionTintOf, restKindOf, type GrenadeRestFx } from '../model/grenadeFx'
import { type CanvasView, projectTo } from '../model/replayView'

/** Fenêtres du calque de fin de vol : la nappe électrique persiste plus que le halo. */
export interface RestWindow {
  frame: number
  /** Rémanence du halo « dernière position connue », en frames. */
  holdHalo: number
  /** Rémanence de la nappe électrique (Shock/Dynamo), en frames. */
  holdDynamo: number
  /** Durée réelle d'une frame, en ms : l'explosion a une timeline en TEMPS, pas en frames
   *  (la cadence d'échantillonnage est choisie au build et peut changer). */
  frameMs: number
}

/**
 * drawGrenadeRestLayer pose l'effet de FIN DE VOL de chaque grenade liée à son projectile.
 *
 * CE QUE LE POINT SIGNIFIE, et le rendu doit le respecter : la DERNIÈRE POSITION RÉPLIQUÉE
 * du projectile, JAMAIS un impact — le film n'enregistre aucune détonation. L'EXPLOSION
 * POSÉE LÀ EST DONC UNE MISE EN SCÈNE ASSUMÉE (item 3.2), et c'est pourquoi l'écran continue
 * de dire « dernière position connue ».
 *
 * PAR TYPE (`restKindOf`) : la Frag, la Plasma et la Spike détonent ; la Shock/Dynamo, elle,
 * n'explose PAS — elle laisse la nappe électrique persistante livrée au lot 2.3, parce que
 * c'est l'effet que l'arme entretient dans le jeu.
 */
export function drawGrenadeRestLayer(
  ctx: CanvasRenderingContext2D,
  fx: GrenadeRestFx[],
  view: CanvasView,
  win: RestWindow,
  style: GrenadeRestStyle,
): void {
  for (const e of fx) {
    const kind = restKindOf(e.rank)
    const hold = kind === 'nappe' ? win.holdDynamo : win.holdHalo
    const age = win.frame - e.frame
    if (age < 0 || age > hold) continue
    const c = projectTo(view, e)
    if (kind === 'explosion') {
      drawExplosion(
        ctx,
        {
          x: c.x,
          y: c.y,
          ageMs: age * win.frameMs,
          seed: e.seed,
          k: style.k,
          reduced: style.reducedMotion,
        },
        {
          fire: style.ink.tint[explosionTintOf(e.rank)] || style.ink.tint.neutral,
          core: style.ink.core,
          smoke: style.smoke,
        },
      )
      continue
    }
    if (kind === 'nappe') {
      // La nappe prend la teinte ÉLECTRIQUE du thème, pas la couleur des lancers.
      ctx.strokeStyle = style.ink.tint.electric || style.halo
      const ageMs = age * win.frameMs
      drawDynamoRest(
        ctx,
        c.x,
        c.y,
        dynamoAlpha(ageMs, hold * win.frameMs),
        ageMs,
        e.seed,
        style.reducedMotion,
      )
      continue
    }
    ctx.strokeStyle = style.halo
    ctx.fillStyle = style.halo
    drawRestHalo(ctx, c.x, c.y, 1 - age / (hold + 1), style.reducedMotion)
  }
  ctx.globalAlpha = 1
}

/** Style du calque de fin de vol : les encres de l'explosion, et celle du halo discret. */
export interface GrenadeRestStyle {
  /** Teintes de nature (fxInk.ts) : la déflagration suit le TYPE, jamais le lanceur. */
  ink: FxInk
  /** Poussière résiduelle : encre de mise en page du thème, pas une couleur de donnée. */
  smoke: string
  /** Couleur du halo « dernière position connue » et de la nappe électrique (inchangée). */
  halo: string
  /** Densité du canevas. */
  k: number
  reducedMotion: boolean
}

/** Halo discret : un anneau qui s'éteint sur place — il ne s'ouvre pas, il n'affirme rien. */
function drawRestHalo(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  fade: number,
  reduced: boolean,
): void {
  const k = reduced ? 0.6 : fade
  ctx.globalAlpha = 0.55 * k
  ctx.lineWidth = 1.4
  ctx.beginPath()
  ctx.arc(x, y, 5.5, 0, Math.PI * 2)
  ctx.stroke()
  ctx.globalAlpha = 0.8 * k
  ctx.beginPath()
  ctx.arc(x, y, 1.6, 0, Math.PI * 2)
  ctx.fill()
}

/** Rayon de la nappe, en pixels — celui de la nappe d'avant : aucune portée n'est mesurée. */
const DYNAMO_RADIUS_PX = 9
/** Nombre d'arcs qui rebondissent sur la bordure invisible (planche R2-4, variante 2). */
const DYNAMO_ARCS = 9
/** Pas d'animation, en ms de match : au-delà, la nappe grésillerait ; en deçà, elle fige. */
const DYNAMO_STEP_MS = 100

/**
 * dynamoAlpha — L'AVANCEMENT VISIBLE de la nappe à cet âge, dans [0, 1].
 *
 * PLATEAU PUIS CHUTE, et c'est la réponse mesurée au retour A4 (« la nappe doit persister au
 * moins 2,5 s au sol »). L'ancienne courbe était une simple droite `1 - âge/fenêtre` : à
 * 2,0 s l'opacité maximale de la nappe valait déjà 0,20, et à 2,1 s elle passait sous le
 * seuil où un trait de 1,2 px se distingue encore d'un fond de carte. La fenêtre DÉCLARÉE
 * disait 2,5 s, l'écran en montrait 2,1.
 *
 * La nappe tient donc son plein éclat sur les trois premiers quarts de la fenêtre, puis
 * s'éteint sur le dernier quart. Combinée à une fenêtre portée à 3,0 s, elle reste lisible
 * au-delà des 2,5 s demandées, et disparaît quand même — un effet qui s'arrête net à mi-course
 * se lit comme une coupure de rendu.
 */
export function dynamoAlpha(ageMs: number, holdMs: number): number {
  if (!(holdMs > 0)) return 0
  const u = Math.max(0, Math.min(1, ageMs / holdMs))
  return u <= 0.75 ? 1 : Math.max(0, 1 - (u - 0.75) / 0.25)
}

/**
 * Nappe électrique de la Shock/Dynamo — VARIANTE 2 DE LA PLANCHE (R2-4), portée telle que
 * l'utilisateur l'a choisie le 2026-08-18 : une NAPPE DIFFUSE, sans anneau, faite d'arcs qui
 * rebondissent sur une bordure invisible. Ce qui disparaît par rapport à l'ancienne : l'anneau
 * net, et les trois arcs qui rayonnaient du centre.
 *
 * ELLE EST FONCTION DE L'ÂGE, jamais de l'horloge murale (patron `explosionFx`) : le même âge
 * rend toujours la même image, donc un retour en arrière rejoue exactement ce qu'on a vu. La
 * planche, elle, animait sur le temps réel — c'était une vignette en boucle, pas un rejeu.
 *
 * TEINTE : `--replay-fx-electric` du thème, celle de la planche. La nappe empruntait jusqu'ici
 * la couleur des LANCERS (token `info`) : c'est l'écart planche/production que ce portage
 * corrige, puisque c'est la nappe électrique que l'utilisateur a validée. Thème sans cette
 * teinte : repli sur l'encre du halo, jamais une couleur inventée.
 *
 * Sous « mouvement réduit », les arcs se figent (pas d'avancement) — la nappe ne disparaît pas.
 */
function drawDynamoRest(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  alpha: number,
  ageMs: number,
  seed: number,
  reduced: boolean,
): void {
  const r = DYNAMO_RADIUS_PX
  // La NAPPE : un dégradé radial qui s'éteint sur le bord — c'est elle qui donne l'emprise.
  const grad = ctx.createRadialGradient(x, y, 0, x, y, r)
  grad.addColorStop(0, ctx.strokeStyle as string)
  grad.addColorStop(1, 'transparent')
  ctx.globalAlpha = 0.2 * alpha
  ctx.fillStyle = grad
  ctx.beginPath()
  ctx.arc(x, y, r, 0, Math.PI * 2)
  ctx.fill()
  // LES ARCS : chacun part du bord, plonge vers l'intérieur et repart au bord — ils
  // « rebondissent » sur une bordure qu'on ne trace pas.
  const pas = reduced ? 0 : Math.floor(ageMs / DYNAMO_STEP_MS)
  ctx.lineWidth = 1
  for (let i = 0; i < DYNAMO_ARCS; i++) {
    // LE GERME ENTRE DANS LA FORMULE : sans lui, deux Dynamos au sol au même instant
    // porteraient exactement les mêmes arcs, et la carte se lirait comme un copier-coller.
    const g = (seed | 0) * 7 + i * 53 + pas * 17
    const a0 = ((g % 360) / 360) * Math.PI * 2
    const a1 = a0 + 1.6 + (g % 7) / 7
    const mid = (a0 + a1) / 2
    const creux = 0.3 + (g % 4) / 10
    ctx.globalAlpha = (0.3 + 0.45 * ((g % 5) / 5)) * alpha
    ctx.beginPath()
    ctx.moveTo(x + Math.cos(a0) * r * 0.92, y + Math.sin(a0) * r * 0.92)
    ctx.lineTo(x + Math.cos(mid) * r * creux, y + Math.sin(mid) * r * creux)
    ctx.lineTo(x + Math.cos(a1) * r * 0.92, y + Math.sin(a1) * r * 0.92)
    ctx.stroke()
  }
}
