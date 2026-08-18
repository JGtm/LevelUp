/**
 * replayDraw.ts — couches de DÉCOR et d'ÉVÉNEMENTS du rejeu 2D : sol reconstruit, props Forge
 * (en repli), tirs et lancers de grenade. Le joueur lui-même vit dans replayMarkers.ts.
 * Pas de React : uniquement un CanvasRenderingContext2D + de la géométrie pure
 * (replayLogic.ts). Les couleurs arrivent DÉJÀ RÉSOLUES depuis les tokens sémantiques
 * (getSeriesColors / resolveToken) — aucun littéral de couleur ici (règle color-tokens).
 */
import type { ReplayBounds, ReplayGrenade, ReplayMapObject } from '@/lib/api/types'

import type { FxInk } from './fxInk'
import { altitudeTint, floorRun, hasEdge, type FloorGrid } from './mapFloor'
import { drawMuzzleFlash } from './muzzleFlash'
import type { ShotFxEntry } from './shotFx'
import { drawDeathMarker, drawShotEffect } from './shotEffects'
import { MELEE_LINK_MAX_M, type KillFxEntry } from './killFx'
import { altitudeRatio, canvasScale, footprint, worldToCanvas } from './replayLogic'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
export interface CanvasView {
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
}

/** Amplitude verticale utilisée pour l'indication d'étage. */
interface ZRange {
  min: number
  max: number
}

// Fond de carte : props Forge de 0,25 m² en moyenne — sans plancher de taille ils sont
// invisibles. Opacités volontairement basses : le sujet du rendu reste les joueurs.
const OBJECT_MIN_PX = 2.5
const OBJECT_ALPHA_LOW = 0.14
const OBJECT_ALPHA_SPAN = 0.24

interface GeometryStyle {
  color: string
  z: ZRange
}

/**
 * drawGeometryLayer dessine les props Forge SOUS les trajectoires : rectangles orientés
 * (ou petits carrés quand l'emprise projetée est sous le seuil de lisibilité).
 * L'opacité monte avec l'altitude : indication d'étage discrète, sans couleur dédiée.
 */
export function drawGeometryLayer(
  ctx: CanvasRenderingContext2D,
  objects: ReplayMapObject[],
  view: CanvasView,
  style: GeometryStyle,
): void {
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  ctx.fillStyle = style.color
  for (const o of objects) {
    ctx.globalAlpha =
      OBJECT_ALPHA_LOW + OBJECT_ALPHA_SPAN * altitudeRatio(o.z ?? 0, style.z.min, style.z.max)
    const corners = footprint(o)
    const wide = (o.dx ?? 0) * scale >= OBJECT_MIN_PX && (o.dy ?? 0) * scale >= OBJECT_MIN_PX
    if (corners.length === 4 && wide) {
      ctx.beginPath()
      corners.forEach((w, i) => {
        const c = worldToCanvas(w, view.bounds, view.width, view.height, view.pad)
        if (i === 0) ctx.moveTo(c.x, c.y)
        else ctx.lineTo(c.x, c.y)
      })
      ctx.closePath()
      ctx.fill()
      continue
    }
    const c = worldToCanvas(o, view.bounds, view.width, view.height, view.pad)
    ctx.fillRect(c.x - OBJECT_MIN_PX / 2, c.y - OBJECT_MIN_PX / 2, OBJECT_MIN_PX, OBJECT_MIN_PX)
  }
  ctx.globalAlpha = 1
}

// Les TRAJECTOIRES et les marqueurs de joueur vivent dans replayMarkers.ts : ce calque a
// gagné le cône de visée, le bouclier, les anneaux d'étage, l'apparition et la mort, et il
// aurait fait de ce fichier un god file.

// Le SOL RECONSTRUIT (cf. mapFloor.ts). L'opacité monte avec l'altitude du sol : les étages se
// distinguent sans qu'aucune couleur ne leur soit dédiée. Bornes reprises du POC, où elles ont
// été réglées à l'écran — assez franches pour que la carte se reconnaisse, assez basses pour
// que les trajectoires restent le sujet.
const FLOOR_ALPHA_LOW = 0.1
const FLOOR_ALPHA_SPAN = 0.46
const FLOOR_EDGE_ALPHA = 0.32

// Événements ponctuels : un tir est un éclair de bouche (sa géométrie vit dans
// muzzleFlash.ts), un lancer une marque plus lisible.
const GRENADE_RADIUS = 4
const GRENADE_RING = 6.5

/** Couleurs du fond de carte : l'aplat du sol et le trait de ses arêtes. */
export interface FloorStyle {
  fill: string
  edge: string
}

/**
 * drawFloorLayer peint le sol reconstruit : aplats par plage d'altitude, puis arêtes.
 *
 * COÛTEUX ET INVARIANT — à peindre UNE FOIS dans un canvas hors écran, puis à recopier à
 * chaque image. La trame fait ~45 000 cellules : la repeindre à 60 images par seconde
 * consommerait tout le budget d'animation pour un fond qui ne bouge pas.
 */
export function drawFloorLayer(
  ctx: CanvasRenderingContext2D,
  grid: FloorGrid,
  view: CanvasView,
  style: FloorStyle,
): void {
  ctx.fillStyle = style.fill
  for (let y = 0; y < grid.ny; y++) {
    // Les bords sont ARRONDIS AU PIXEL : deux plages voisines qui se chevaucheraient d'un
    // pixel dessineraient un quadrillage parasite sur un sol continu.
    const yTop = Math.round(cellY(grid, y + 1, view))
    const height = Math.max(1, Math.round(cellY(grid, y, view)) - yTop)
    let x = 0
    while (x < grid.nx) {
      const run = floorRun(grid, x, y)
      if (run === 0) {
        x++
        continue
      }
      const xLeft = Math.round(cellX(grid, x, view))
      const xRight = Math.round(cellX(grid, x + run, view))
      ctx.globalAlpha =
        FLOOR_ALPHA_LOW + FLOOR_ALPHA_SPAN * altitudeTint(grid.topZ[y * grid.nx + x], grid)
      ctx.fillRect(xLeft, yTop, Math.max(1, xRight - xLeft), height)
      x += run
    }
  }
  ctx.globalAlpha = 1
  drawFloorEdges(ctx, grid, view, style.edge)
}

/**
 * drawFloorEdges trace les marches et bords de vide. Un seul chemin pour toute la carte : des
 * milliers de `stroke()` séparés coûteraient bien plus que le tracé lui-même.
 */
function drawFloorEdges(
  ctx: CanvasRenderingContext2D,
  grid: FloorGrid,
  view: CanvasView,
  color: string,
): void {
  ctx.strokeStyle = color
  ctx.globalAlpha = FLOOR_EDGE_ALPHA
  ctx.lineWidth = 1
  ctx.beginPath()
  for (let y = 0; y < grid.ny; y++) {
    for (let x = 0; x < grid.nx; x++) {
      // Le demi-pixel place le trait SUR la grille de pixels plutôt qu'à cheval, sans quoi un
      // trait de 1 px s'étale sur deux et paraît flou.
      const xL = Math.round(cellX(grid, x, view)) + 0.5
      const xR = Math.round(cellX(grid, x + 1, view)) + 0.5
      const yT = Math.round(cellY(grid, y + 1, view)) + 0.5
      const yB = Math.round(cellY(grid, y, view)) + 0.5
      if (hasEdge(grid, x, y, 'right')) {
        ctx.moveTo(xR, yT)
        ctx.lineTo(xR, yB)
      }
      if (hasEdge(grid, x, y, 'up')) {
        ctx.moveTo(xL, yT)
        ctx.lineTo(xR, yT)
      }
    }
  }
  ctx.stroke()
  ctx.globalAlpha = 1
}

function cellX(grid: FloorGrid, x: number, view: CanvasView): number {
  const w = { x: grid.minX + x * grid.cell, y: 0 }
  return worldToCanvas(w, view.bounds, view.width, view.height, view.pad).x
}

function cellY(grid: FloorGrid, y: number, view: CanvasView): number {
  const w = { x: 0, y: grid.minY + y * grid.cell }
  return worldToCanvas(w, view.bounds, view.width, view.height, view.pad).y
}

/** Fenêtre d'affichage d'un événement ponctuel, en frames. */
export interface EventWindow {
  frame: number
  /** Nombre de frames pendant lesquelles l'événement reste visible après son instant. */
  hold: number
}

/**
 * drawShotsLayer dessine les ÉCLAIRS DE BOUCHE de la fenêtre courante.
 *
 * CE QUE LE POINT SIGNIFIE, et il faut que le rendu le respecte : le film n'enregistre que les
 * tirs qui INFLIGENT un dégât. Un tir dessiné a donc touché. Sa DIRECTION est celle du REGARD
 * du tireur à cet instant (relu dans sa trajectoire, cf. shotFx.ts) — sans lecture, une
 * bouffée ronde, jamais une direction inventée.
 *
 * CE CALQUE A CHANGÉ DE NATURE le 2026-08-15 (étape 2 du plan des effets de tirs) : il
 * dessinait une TRACE de 62 px dans la couleur du TIREUR ; il dessine désormais un éclair
 * court à la bouche de l'arme, teinté par la NATURE DE LA DÉCHARGE et par elle seule
 * (décision utilisateur). Le cône de visée, lui, n'a pas bougé — le flash s'y ajoute.
 */
export function drawShotsLayer(
  ctx: CanvasRenderingContext2D,
  shots: ShotFxEntry[],
  view: CanvasView,
  win: EventWindow,
  style: ShotStyle,
): void {
  for (const s of shots) {
    const age = win.frame - s.frame
    if (age < 0 || age > win.hold) continue
    const c = worldToCanvas(s, view.bounds, view.width, view.height, view.pad)
    drawMuzzleFlash(ctx, s.fam, s.tint, {
      x: c.x,
      y: c.y,
      // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi.
      angle: s.h === null ? null : (-s.h * Math.PI) / 180,
      fade: 1 - age / Math.max(win.hold, 1),
      reduced: style.reducedMotion,
      seed: s.seed,
      k: style.k,
    }, style.ink)
  }
  ctx.globalAlpha = 1
}

/** Style du calque des tirs : les encres du thème et la densité de l'écran. */
export interface ShotStyle {
  /**
   * Teintes de décharge résolues depuis le thème (fxInk.ts). AUCUNE couleur de joueur
   * n'entre ici : correction utilisateur du 2026-08-15 — « les couleurs des effets de tirs
   * [...] prennent seulement l'ARME en compte ».
   */
  ink: FxInk
  /** Densité du canevas : l'éclair s'adresse à l'œil, sa taille est en pixels d'écran. */
  k: number
  reducedMotion: boolean
}

/** Style du calque des morts : la couleur du tueur, et le repli quand il n'a pas de trace. */
export interface KillFxStyle {
  colorOfSlot: (slot: number) => string | null
  fallback: string
  reducedMotion: boolean
}

/**
 * drawKillFxLayer dessine les EFFETS DE MORT de la fenêtre courante.
 *
 * ORIENTÉ TUEUR -> VICTIME seulement quand le couple est complet (règle POC 89/93, portée
 * par `buildKillFx` : `vx` null = pas d'axe) ; sinon un marqueur pointillé non orienté.
 * L'extrémité est RÉELLE (`target`) : c'est la seule différence de nature avec les tirs,
 * dont la longueur n'est qu'une trace. La COULEUR est celle du tueur — la famille d'arme
 * se lit dans la FORME (arbitrage du lot 3.2, conservé).
 */
export function drawKillFxLayer(
  ctx: CanvasRenderingContext2D,
  fx: KillFxEntry[],
  view: CanvasView,
  win: EventWindow,
  style: KillFxStyle,
): void {
  for (const e of fx) {
    const age = win.frame - e.frame
    if (age < 0 || age > win.hold) continue
    const fade = 1 - age / (win.hold + 1)
    const c = worldToCanvas(e, view.bounds, view.width, view.height, view.pad)
    const color = (e.slot !== null ? style.colorOfSlot(e.slot) : null) ?? style.fallback
    let angle: number | null = null
    let length = 0
    if (e.vx !== null && e.vy !== null) {
      const v = worldToCanvas({ x: e.vx, y: e.vy }, view.bounds, view.width, view.height, view.pad)
      const dx = v.x - c.x
      const dy = v.y - c.y
      const d = Math.hypot(dx, dy)
      // < 1,5 px : tueur et victime au même endroit à l'écran (corps à corps). Aucun axe
      // fiable à en tirer — le marqueur non orienté vaut mieux qu'un angle calculé sur du
      // bruit d'arrondi (règle POC).
      if (d > 1.5) {
        angle = Math.atan2(dy, dx)
        length = d
      }
    }
    const shape = {
      x: c.x,
      y: c.y,
      angle,
      length,
      fade,
      reduced: style.reducedMotion,
      seed: e.seed,
      target: angle !== null,
      meleeLink: e.dist !== null && e.dist < MELEE_LINK_MAX_M,
    }
    if (angle === null) drawDeathMarker(ctx, shape, color)
    else drawShotEffect(ctx, e.fam, shape, color)
  }
  ctx.globalAlpha = 1
}

/** Côté de la vignette de type posée au-dessus de l'anneau d'un lancer (POC : 18 px). */
const GRENADE_ICON_PX = 18

/**
 * tintedIconCanvas — un masque du HUD (blanc/gris + alpha) TEINT à une encre du thème,
 * une fois pour toutes dans un canvas hors écran. Un canvas ne connaît pas le
 * `mask-image` CSS de WeaponIcon : la teinte se fait par composition `source-in`, qui
 * préserve l'alpha et suit le thème par re-teinture (l'appelant re-teint au changement).
 */
export function tintedIconCanvas(img: HTMLImageElement, color: string): HTMLCanvasElement {
  const off = document.createElement('canvas')
  off.width = Math.max(1, img.naturalWidth)
  off.height = Math.max(1, img.naturalHeight)
  const octx = off.getContext('2d')
  if (!octx) return off
  octx.drawImage(img, 0, 0)
  octx.globalCompositeOperation = 'source-in'
  octx.fillStyle = color
  octx.fillRect(0, 0, off.width, off.height)
  return off
}

/** Style du calque des lancers : la couleur des marques, et la vignette du TYPE par rang. */
export interface GrenadeStyle {
  color: string
  /** Vignette teintée du rang, ou null : l'anneau seul reste juste — jamais la vignette
   *  d'un type voisin. */
  iconOf: (rank: number) => CanvasImageSource | null
}

/**
 * drawGrenadesLayer dessine les lancers de grenade.
 *
 * CE QUI EST DESSINÉ EST LE POINT DE DÉPART, pas une trajectoire : l'arc et le point de chute
 * ne sont pas décodés, et rien ici ne les invente. L'anneau distingue le lancer d'un tir ;
 * la VIGNETTE au-dessus dit le TYPE (item 2.4 — le rang est écrit dans le film, la table
 * grenadeLabels le nomme).
 */
export function drawGrenadesLayer(
  ctx: CanvasRenderingContext2D,
  grenades: ReplayGrenade[],
  view: CanvasView,
  win: EventWindow,
  style: GrenadeStyle,
): void {
  ctx.strokeStyle = style.color
  ctx.fillStyle = style.color
  for (const g of grenades) {
    const age = win.frame - g.t
    if (age < 0 || age > win.hold) continue
    const fade = 1 - age / Math.max(win.hold, 1)
    const c = worldToCanvas(g, view.bounds, view.width, view.height, view.pad)
    ctx.globalAlpha = fade
    ctx.beginPath()
    ctx.arc(c.x, c.y, GRENADE_RADIUS, 0, Math.PI * 2)
    ctx.fill()
    ctx.beginPath()
    ctx.arc(c.x, c.y, GRENADE_RING, 0, Math.PI * 2)
    ctx.lineWidth = 1.5
    ctx.stroke()
    const icon = style.iconOf(g.rank)
    if (icon) {
      ctx.globalAlpha = Math.min(1, 1.2 * fade)
      ctx.drawImage(
        icon,
        c.x - GRENADE_ICON_PX / 2,
        c.y - GRENADE_ICON_PX / 2 - 13,
        GRENADE_ICON_PX,
        GRENADE_ICON_PX,
      )
    }
  }
  ctx.globalAlpha = 1
}
