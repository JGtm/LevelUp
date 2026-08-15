/**
 * replayMarkers.ts — LE JOUEUR SUR LA CARTE : traînée, marqueur d'étage, cône de visée,
 * apparition et mort.
 *
 * CE QUE CE CALQUE NE DESSINE PLUS. Une jauge de BOUCLIER surmontait chaque marqueur ; elle a
 * été retirée le 2026-08-15 à la demande de l'utilisateur — « je veux pas ces barres
 * horizontales au dessus des points de joueurs » — et cette demande remplace la décision du
 * 2026-08-13 qui l'avait rendue permanente. Le champ `sh` continue de voyager : ce sont les
 * FICHES de joueurs qui le lisent (rosterLogic.ts). C'est le dessin sur la carte qui est
 * refusé, pas la mesure.
 *
 * Une trace = UNE VIE, jamais un joueur : le slot de biped est réattribué à chaque
 * réapparition. Les trois instants d'une vie se lisent donc séparément — elle S'OUVRE (anneau),
 * elle DURE (marqueur + traînée), elle SE FERME (croix). Sans ces repères, une vie qui
 * disparaît et une autre qui commence deux mètres plus loin sont indiscernables.
 *
 * TOUT CE QUI S'ADRESSE À L'ŒIL EST À L'ÉCHELLE DE L'ÉCRAN, pas du canevas. Le canevas est
 * rendu à la densité du périphérique : un rayon déclaré en pixels de canevas apparaîtrait deux
 * fois plus petit sur un écran à forte densité. Les positions, elles, appartiennent au MONDE et
 * n'y touchent pas. C'est la distinction que le POC avait dû introduire après avoir constaté
 * que des points « grossis » restaient invisibles.
 *
 * Aucun littéral de couleur : les teintes de trace arrivent résolues depuis les tokens, les
 * encres de lisibilité depuis le thème (cf. canvasInk.ts).
 */
import type { ReplayProjectileReady, ReplayTrackReady } from './replayNormalize'

import {
  altitudeAt,
  floorOf,
  freshness,
  heldReading,
  isAliveAt,
  positionAt,
  trackWindow,
  trailAt,
  worldToCanvas,
  type XY,
} from './replayLogic'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
export interface CanvasView {
  bounds: { minX: number; minY: number; maxX: number; maxY: number }
  width: number
  height: number
  pad: number
}

// --- Durées, en frames, converties par l'appelant depuis le temps réel ---------------------

/**
 * Réglages temporels du calque, tous exprimés en frames de la grille du rejeu.
 * Les valeurs de référence viennent du POC, où elles ont été réglées à l'écran :
 * traînée 7 s, cône maintenu 5 s (la couverture de la visée passe de 43,6 % à 93,5 % du temps
 * de jeu), mort marquée 1,5 s, apparition 0,8 s.
 */
export interface MarkerTiming {
  trail: number
  aimHold: number
  death: number
  spawn: number
}

// --- Constantes de rendu (pixels d'ÉCRAN, sauf mention) ------------------------------------

const TRAIL_ALPHA = 0.55
const TRAIL_WIDTH = 1.9

/** Rayon du noyau du marqueur, et son accroissement par étage. */
const CORE_RADIUS = 4.6
const CORE_PER_FLOOR = 0.9
/** Halo diffus sous le marqueur : il détache le joueur du sol sans masquer la carte. */
const HALO_RADIUS = 9
const HALO_PER_FLOOR = 2
const HALO_ALPHA = 0.15
/** Anneaux concentriques : un par étage au-dessus du sol. */
const RING_GAP = 3.6
const RING_ALPHA = 0.85
const RING_ALPHA_DECAY = 0.18
/** Liseré de lisibilité : la carte va du clair au sombre, un point coloré s'y perd sans lui. */
const OUTLINE_PAD = 1.15
const OUTLINE_ALPHA = 0.62

const AIM_LENGTH = 46
const AIM_HALF_ANGLE = 0.3
const AIM_CONE_ALPHA = 0.44
const AIM_AXIS_ALPHA = 0.92
const AIM_AXIS_WIDTH = 1.8
const AIM_UNDERLINE_ALPHA = 0.34
const AIM_UNDERLINE_WIDTH = 3.6
/** Une visée de 5 s ne vaut pas une visée de l'instant : elle perd 62 % de son opacité. */
const AIM_FADE = 0.62

const DEATH_RADIUS = 5
const DEATH_GROWTH = 7
const DEATH_ALPHA = 0.85
const SPAWN_RADIUS = 6
const SPAWN_GROWTH = 14
const SPAWN_ALPHA = 0.8

const PROJECTILE_ALPHA = 0.5
const PROJECTILE_WIDTH = 1.2
/** Le vol reste visible brièvement après son dernier point répliqué, puis s'efface. */
const PROJECTILE_TAIL_FRAMES = 7

/** Style du calque : une couleur par trace, plus les encres de lisibilité du thème. */
export interface MarkerStyle {
  colors: string[]
  /** Encre qui contraste avec la page dans les deux thèmes : liseré, piste de bouclier. */
  ink: string
  frame: number
  timing: MarkerTiming
  z: { min: number; max: number }
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
  showAim: boolean
}

/**
 * drawTracksLayer dessine chaque vie à la frame courante : celles qui vivent, et celles qui
 * viennent de se fermer (la croix survit 1,5 s à la vie qu'elle termine).
 */
export function drawTracksLayer(
  ctx: CanvasRenderingContext2D,
  tracks: ReplayTrackReady[],
  view: CanvasView,
  style: MarkerStyle,
): void {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  tracks.forEach((track, i) => {
    const color = style.colors[i] ?? style.colors[0]
    if (!color) return
    if (!isAliveAt(track, style.frame)) {
      drawDeathMark(ctx, track, view, style, color)
      return
    }
    drawLivingTrack(ctx, track, view, style, color)
  })
  ctx.globalAlpha = 1
}

/**
 * drawDeathMark marque l'endroit de la dernière position transmise pendant `timing.death`,
 * puis plus rien : le joueur a disparu jusqu'à sa réapparition, qui est une AUTRE vie.
 */
function drawDeathMark(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: MarkerStyle,
  color: string,
): void {
  const age = style.frame - trackWindow(track).end
  if (age < 0 || age > style.timing.death) return
  const last = track.points[track.points.length - 1]
  if (!last) return
  const c = project(last, view)
  const fade = 1 - age / style.timing.death
  const r = (DEATH_RADIUS + DEATH_GROWTH * (1 - fade)) * style.k
  ctx.strokeStyle = color
  ctx.globalAlpha = DEATH_ALPHA * fade
  ctx.lineWidth = 2 * style.k
  ctx.beginPath()
  ctx.moveTo(c.x - r, c.y - r)
  ctx.lineTo(c.x + r, c.y + r)
  ctx.moveTo(c.x + r, c.y - r)
  ctx.lineTo(c.x - r, c.y + r)
  ctx.stroke()
  ctx.globalAlpha = 1
}

/** drawLivingTrack : traînée, cône, apparition, marqueur d'étage. */
function drawLivingTrack(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: MarkerStyle,
  color: string,
): void {
  const head = positionAt(track.points, style.frame)
  if (!head) return
  const c = project(head, view)
  const fl = floorIndex(track, style)

  drawTrail(ctx, track, view, style, color)
  if (style.showAim) drawAimCone(ctx, track, c, style, color)
  drawSpawnRing(ctx, track, c, style, color)
  drawMarker(ctx, c, style, color, fl)
}

function drawTrail(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: MarkerStyle,
  color: string,
): void {
  const trail = trailAt(track.points, style.frame, style.timing.trail)
  if (trail.length < 2) return
  ctx.beginPath()
  trail.forEach((p, k) => {
    const c = project(p, view)
    if (k === 0) ctx.moveTo(c.x, c.y)
    else ctx.lineTo(c.x, c.y)
  })
  ctx.strokeStyle = color
  ctx.globalAlpha = TRAIL_ALPHA
  ctx.lineWidth = TRAIL_WIDTH * style.k
  ctx.stroke()
  ctx.globalAlpha = 1
}

/**
 * drawAimCone dessine la DIRECTION DU REGARD, décodée du même record que la position.
 *
 * Le cône se dégrade du centre vers le bord — dense à l'origine, où il faut lire QUI vise,
 * transparent au bout, où il ne faut pas masquer le décor. Il pâlit avec l'âge de la mesure et
 * n'est PAS dessiné au-delà du maintien : passé ce délai, on ne sait plus où le joueur regarde,
 * et une direction périmée affirmerait ce qu'on ignore.
 */
function drawAimCone(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: MarkerStyle,
  color: string,
): void {
  const read = heldReading(track.points, style.frame, (p) => p.h, style.timing.aimHold)
  if (!read) return
  const fresh = freshness(read.age, style.timing.aimHold, AIM_FADE)
  // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi.
  const ang = (-read.value * Math.PI) / 180
  const R = AIM_LENGTH * style.k
  const gradient = ctx.createRadialGradient(c.x, c.y, 0, c.x, c.y, R)
  gradient.addColorStop(0, color)
  gradient.addColorStop(1, 'transparent')
  ctx.globalAlpha = AIM_CONE_ALPHA * fresh
  ctx.beginPath()
  ctx.moveTo(c.x, c.y)
  ctx.arc(c.x, c.y, R, ang - AIM_HALF_ANGLE, ang + AIM_HALF_ANGLE)
  ctx.closePath()
  ctx.fillStyle = gradient
  ctx.fill()

  // L'axe porte deux traits : un liseré d'encre dessous, la couleur de la trace dessus. Sans le
  // liseré, un cône clair sur une zone claire de la carte disparaît complètement.
  const ex = c.x + Math.cos(ang) * R
  const ey = c.y + Math.sin(ang) * R
  strokeAxis(ctx, c, ex, ey, style.ink, AIM_UNDERLINE_ALPHA * fresh, AIM_UNDERLINE_WIDTH * style.k)
  strokeAxis(ctx, c, ex, ey, color, AIM_AXIS_ALPHA * fresh, AIM_AXIS_WIDTH * style.k)
  ctx.globalAlpha = 1
}

function strokeAxis(
  ctx: CanvasRenderingContext2D,
  from: XY,
  ex: number,
  ey: number,
  color: string,
  alpha: number,
  width: number,
): void {
  ctx.beginPath()
  ctx.moveTo(from.x, from.y)
  ctx.lineTo(ex, ey)
  ctx.strokeStyle = color
  ctx.globalAlpha = alpha
  ctx.lineWidth = width
  ctx.stroke()
}

/** drawSpawnRing : l'anneau qui s'ouvre au premier instant de la vie. */
function drawSpawnRing(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: MarkerStyle,
  color: string,
): void {
  const age = style.frame - trackWindow(track).start
  if (age < 0 || age > style.timing.spawn) return
  const f = 1 - age / style.timing.spawn
  ctx.beginPath()
  ctx.arc(c.x, c.y, (SPAWN_RADIUS + SPAWN_GROWTH * (1 - f)) * style.k, 0, Math.PI * 2)
  ctx.strokeStyle = color
  ctx.globalAlpha = SPAWN_ALPHA * f
  ctx.lineWidth = 2 * style.k
  ctx.stroke()
  ctx.globalAlpha = 1
}

/**
 * drawMarker dessine le joueur : halo, anneaux d'étage, liseré, noyau.
 *
 * L'ÉTAGE SE LIT PAR DES ANNEAUX CONCENTRIQUES, jamais par un décalage du marqueur : en vue de
 * dessus, déplacer un point vers le haut de l'écran voudrait dire « plus au nord » et
 * fausserait la position. L'altitude est un palier, pas un dégradé — l'histogramme des z montre
 * des pics nets, la carte a trois niveaux de jeu.
 */
function drawMarker(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: MarkerStyle,
  color: string,
  fl: number,
): void {
  const core = (CORE_RADIUS + CORE_PER_FLOOR * fl) * style.k
  ctx.fillStyle = color
  ctx.globalAlpha = HALO_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, (HALO_RADIUS + HALO_PER_FLOOR * fl) * style.k, 0, Math.PI * 2)
  ctx.fill()

  ctx.strokeStyle = color
  ctx.lineWidth = 1.5 * style.k
  for (let r = 1; r <= fl; r++) {
    ctx.globalAlpha = RING_ALPHA - RING_ALPHA_DECAY * (r - 1)
    ctx.beginPath()
    ctx.arc(c.x, c.y, core + RING_GAP * style.k * r, 0, Math.PI * 2)
    ctx.stroke()
  }

  ctx.globalAlpha = OUTLINE_ALPHA
  ctx.fillStyle = style.ink
  ctx.beginPath()
  ctx.arc(c.x, c.y, core + OUTLINE_PAD * style.k, 0, Math.PI * 2)
  ctx.fill()

  ctx.globalAlpha = 1
  ctx.fillStyle = color
  ctx.beginPath()
  ctx.arc(c.x, c.y, core, 0, Math.PI * 2)
  ctx.fill()
}

/**
 * drawProjectilesLayer dessine les vols de projectile en cours.
 *
 * LE DERNIER POINT N'EST PAS UN IMPACT : le film ne porte aucun événement de détonation. C'est
 * la dernière position RÉPLIQUÉE — pour une grenade, la réplication cesse ~1,4 s après le
 * lancer alors que la mèche court jusqu'à ~3 s. Le vol s'efface donc, il n'explose pas.
 */
export function drawProjectilesLayer(
  ctx: CanvasRenderingContext2D,
  projectiles: ReplayProjectileReady[],
  view: CanvasView,
  frame: number,
  color: string,
): void {
  ctx.strokeStyle = color
  ctx.lineWidth = PROJECTILE_WIDTH
  for (const pr of projectiles) {
    const pts = pr.p
    if (pts.length < 2) continue
    const end = pr.t0 + pts[pts.length - 1][0]
    if (frame < pr.t0 || frame > end + PROJECTILE_TAIL_FRAMES) continue
    const fade = frame > end ? 1 - (frame - end) / PROJECTILE_TAIL_FRAMES : 1
    ctx.globalAlpha = PROJECTILE_ALPHA * fade
    ctx.beginPath()
    let started = false
    for (const [dt, x, y] of pts) {
      if (pr.t0 + dt > frame) break
      const c = project({ x, y }, view)
      if (!started) {
        ctx.moveTo(c.x, c.y)
        started = true
      } else ctx.lineTo(c.x, c.y)
    }
    if (started) ctx.stroke()
  }
  ctx.globalAlpha = 1
}

function project(p: XY, view: CanvasView): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

function floorIndex(track: ReplayTrackReady, style: MarkerStyle): number {
  const z = altitudeAt(track.points, style.frame)
  return z === null ? 0 : floorOf(z, style.z.min, style.z.max)
}
