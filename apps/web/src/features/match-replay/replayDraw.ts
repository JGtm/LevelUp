/**
 * replayDraw.ts — couches de DÉCOR et d'ÉVÉNEMENTS du rejeu 2D : sol reconstruit, props Forge
 * (en repli), tirs et lancers de grenade. Le joueur lui-même vit dans replayMarkers.ts.
 * Pas de React : uniquement un CanvasRenderingContext2D + de la géométrie pure
 * (replayLogic.ts). Les couleurs arrivent DÉJÀ RÉSOLUES depuis les tokens sémantiques
 * (getSeriesColors / resolveToken) — aucun littéral de couleur ici (règle color-tokens).
 */
import type { ReplayBounds, ReplayGrenade, ReplayMapObject, ReplayShot } from '@/lib/api/types'

import { altitudeTint, floorRun, hasEdge, type FloorGrid } from './mapFloor'
import { drawShotEffect, familyOf } from './shotEffects'
import { altitudeRatio, canvasScale, footprint, worldToCanvas } from './replayLogic'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
interface CanvasView {
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

// Événements ponctuels : un tir est un éclat bref, un lancer une marque plus lisible.
// Longueur de la forme d un tir, en pixels : assez pour lire une direction, assez court
// pour ne pas traverser la carte.
const SHOT_LENGTH = 26
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
 * drawShotsLayer dessine les tirs de la fenêtre courante.
 *
 * CE QUE LE POINT SIGNIFIE, et il faut que le rendu le respecte : le film n'enregistre que les
 * tirs qui INFLIGENT un dégât. Un tir dessiné a donc touché. Sa DIRECTION n'est tracée que
 * lorsqu'elle est lisible — sinon on ne dessine que l'éclat, jamais une direction inventée.
 */
export function drawShotsLayer(
  ctx: CanvasRenderingContext2D,
  shots: ReplayShot[],
  view: CanvasView,
  win: EventWindow,
  style: ShotStyle,
): void {
  for (const s of shots) {
    const age = win.frame - s.t
    if (age < 0 || age > win.hold) continue
    const c = worldToCanvas(s, view.bounds, view.width, view.height, view.pad)
    // LA COULEUR EST CELLE DU TIREUR quand on la connaît : c'est elle qui permet de suivre un
    // joueur des yeux. La FAMILLE d'arme, elle, se lit dans la FORME de l'effet.
    const color = style.colorOfSlot(s.slot) ?? style.fallback
    // `w` est le nom du champ AU CONTRAT (identifiant d'arme 64 bits en hexadécimal).
    // L'interface écrite à la main disait `weapon` : le champ était donc toujours
    // indéfini et toutes les familles d'arme retombaient sur la forme par défaut.
    drawShotEffect(ctx, familyOf(style.labelOf(s.w)), {
      x: c.x,
      y: c.y,
      // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi. Absent = pas de
      // visée lisible, et alors aucune direction n'est dessinée.
      angle: s.h === undefined ? null : (-s.h * Math.PI) / 180,
      length: SHOT_LENGTH,
      fade: 1 - age / Math.max(win.hold, 1),
      reduced: style.reducedMotion,
      seed: s.t + s.slot,
    }, color)
  }
  ctx.globalAlpha = 1
}

/** Style du calque des tirs : de quoi retrouver la couleur du tireur et le nom de son arme. */
export interface ShotStyle {
  /** Couleur résolue du slot tireur ; null quand la trace n'est pas identifiée. */
  colorOfSlot: (slot: number) => string | null
  /** Couleur employée quand le tireur n'a pas de trace connue. */
  fallback: string
  /** Nom canonique de l'arme, ou undefined si l'identifiant n'est pas au catalogue. */
  labelOf: (weaponId: string | undefined) => string | undefined
  reducedMotion: boolean
}

/**
 * drawGrenadesLayer dessine les lancers de grenade.
 *
 * CE QUI EST DESSINÉ EST LE POINT DE DÉPART, pas une trajectoire : l'arc et le point de chute
 * ne sont pas décodés, et rien ici ne les invente. L'anneau distingue le lancer d'un tir.
 */
export function drawGrenadesLayer(
  ctx: CanvasRenderingContext2D,
  grenades: ReplayGrenade[],
  view: CanvasView,
  win: EventWindow,
  color: string,
): void {
  ctx.strokeStyle = color
  ctx.fillStyle = color
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
  }
  ctx.globalAlpha = 1
}
