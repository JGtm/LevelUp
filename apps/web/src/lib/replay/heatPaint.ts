/**
 * heatPaint.ts — LE NOYAU PARTAGÉ DE LA CARTE DE CHALEUR (rejeu 2D + analyse tactique).
 *
 * POURQUOI (2026-09-07, Q7 orchestration / S.2 Tactique) : `features/tactical/heatPaint.ts`
 * était une COPIE de `features/match-replay/layers/heatmapLayer.ts` (rampe, étalonnage,
 * tracé canvas À L'IDENTIQUE) — seule différait la FABRICATION (le rejeu ACCUMULE des
 * positions brutes et les LISSE ; le tactique reçoit une grille DÉJÀ agrégée par le
 * serveur). La règle du dépôt (≤ 2 copies) tolérait ça tout juste ; une troisième aurait
 * divergé en silence. Le noyau vit donc ICI ; chaque domaine n'en garde qu'un adaptateur
 * (`layers/useReplayStaticLayers.ts`, `TacticalPlanCard.tsx`).
 *
 * COMMUN : rampe (`heatRamp`), étalonnage [0,1] (`cellIntensity`), TRACÉ (`drawHeatmap`).
 * DIFFÉRENT : la fabrication et la PROJECTION monde -> canvas (Y inversé côté rejeu, pas
 * côté tactique) — abstraite derrière `HeatSource`/`HeatGeometry`, fournis par
 * `drawHeatmapLayer` / `drawTacticalHeatmap`. `k` reste un réglage de style : dpr côté
 * rejeu, 1 côté tactique (flou déjà consigné, `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md`).
 *
 * Pas de React : logique pure + un CanvasRenderingContext2D (même règle que replayDraw).
 */
import { hexToRgba } from '@/components/charts/_utils'
import type { ReplayBounds } from '@/lib/api/types'

import { frameToMs, msToFrames, type XY } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

// ---------------------------------------------------------------------------------------
// Rampe de couleur — IDENTIQUE pour les deux domaines.
// ---------------------------------------------------------------------------------------

/** Paliers de la rampe précalculée : un `rgba()` par palier, indexé pendant le dessin. — color-allow: 2026-09-06 (ronde 2, N1) — ligne de PROSE qui DECRIT la rampe du theme, elle n'en pose aucune. */
export const HEAT_RAMP_STEPS = 64

/** Opacités des extrémités (A8, 2026-08-18 : plafond 0,55 -> 0,75, mesuré 5x plus efficace
 *  que d'abaisser le quantile bas). */
const HEAT_ALPHA_MIN = 0.12
const HEAT_ALPHA_MAX = 0.75

/** Un composant RVB lu d'un hex résolu. */
interface Rgb {
  r: number
  g: number
  b: number
}

function parseHex(hex: string): Rgb | null {
  const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex.trim())
  if (!m) return null
  return { r: parseInt(m[1], 16), g: parseInt(m[2], 16), b: parseInt(m[3], 16) }
}

function mixHex(a: Rgb, b: Rgb, t: number): string {
  const mix = (u: number, v: number) => Math.round(u + (v - u) * t).toString(16).padStart(2, '0')
  return `#${mix(a.r, b.r)}${mix(a.g, b.g)}${mix(a.b, b.b)}`
}

/**
 * heatRamp précalcule la rampe en `rgba()` : couleur ET opacité montent ensemble. N POINTS
 * (2026-08-18), déjà résolus par l'appelant (règle color-tokens), uniformément répartis —
 * deux arrêts donnent une rampe simple, trois le bleu -> rouge -> violet retenu. Un arrêt
 * illisible, ou moins de deux, rend une rampe VIDE plutôt qu'une couleur inventée.
 */
export function heatRamp(stops: readonly string[]): string[] {
  const rgb = stops.map(parseHex)
  if (rgb.length < 2 || rgb.some((c) => c === null)) return []
  const points = rgb as Rgb[]
  const segments = points.length - 1
  const out: string[] = []
  for (let i = 0; i < HEAT_RAMP_STEPS; i++) {
    const t = i / (HEAT_RAMP_STEPS - 1)
    // Segment courant et avancement DANS ce segment : `t = 1` retombe sur le dernier.
    const seg = Math.min(segments - 1, Math.floor(t * segments))
    const u = t * segments - seg
    const alpha = HEAT_ALPHA_MIN + (HEAT_ALPHA_MAX - HEAT_ALPHA_MIN) * t
    out.push(hexToRgba(mixHex(points[seg], points[seg + 1], u), Number(alpha.toFixed(3))))
  }
  return out
}

// ---------------------------------------------------------------------------------------
// Tracé — LE NOYAU COMMUN : fusion des plages de même palier, bords alignés au pixel k.
// ---------------------------------------------------------------------------------------

/** Bas et haut de la rampe d'une grille : sous `lo`, palier 0 ; au-dessus de `hi`, saturé. */
interface HeatScale {
  lo: number
  hi: number
}

/** cellIntensity rend la position d'une valeur sur [0, 1], ou null si jamais atteinte
 *  (valeur ≤ 0) — une cellule jamais vue reste VIDE plutôt que froide. */
function cellIntensity(value: number, scale: HeatScale): number | null {
  if (!(value > 0)) return null
  const span = scale.hi - scale.lo
  if (!(span > 0)) return 1
  const t = (value - scale.lo) / span
  return t < 0 ? 0 : t > 1 ? 1 : t
}

/** Ce que `drawHeatmap` lit d'une grille (dense ou éparse) ; `filled` à 0 = rien à peindre. */
interface HeatSource {
  nx: number
  ny: number
  filled: number
  scale: HeatScale
  /** Valeur brute de la cellule (row, col) — 0 ou absente si jamais atteinte. */
  valueAt(row: number, col: number): number
}

/** Ce que `drawHeatmap` projette (bords physiques, avant `style.k`) — ici divergent le
 *  Y-flip du rejeu et son absence côté tactique. */
interface HeatGeometry {
  /** Bord GAUCHE en pixels canvas de la colonne `col` (bord droit = xAt(col + 1)). */
  xAt(col: number): number
  rowPixels(row: number): { top: number; bottom: number }
}

/** Style du calque : rampe DÉJÀ résolue, et le rapport de pixels de l'écran (`k`). */
export interface HeatmapStyle {
  ramp: readonly string[]
  k: number
}

/**
 * drawHeatmap peint la grille. DEUX PRÉCAUTIONS CONTRE LE QUADRILLAGE PARASITE : les
 * cellules voisines de même palier sont peintes d'un SEUL rectangle, et chaque bord est
 * aligné sur un pixel PHYSIQUE — sans quoi deux rectangles translucides partagent un pixel
 * anti-crénelé et tracent une couture. Calque STATIQUE, cuit hors écran par l'appelant.
 */
function drawHeatmap(
  ctx: CanvasRenderingContext2D,
  source: HeatSource,
  geometry: HeatGeometry,
  style: HeatmapStyle,
): void {
  const last = style.ramp.length - 1
  if (source.filled === 0 || last < 0) return
  const snap = (v: number) => Math.round(v * style.k) / style.k
  const idxAt = (row: number, col: number): number => {
    const t = cellIntensity(source.valueAt(row, col), source.scale)
    return t === null ? -1 : Math.round(t * last)
  }
  for (let row = 0; row < source.ny; row++) {
    const { top, bottom } = geometry.rowPixels(row)
    const y0 = snap(top)
    const y1 = snap(bottom)
    let col = 0
    while (col < source.nx) {
      const idx = idxAt(row, col)
      if (idx < 0) {
        col++
        continue
      }
      let end = col
      while (end + 1 < source.nx && idxAt(row, end + 1) === idx) end++
      const x0 = snap(geometry.xAt(col))
      const x1 = snap(geometry.xAt(end + 1))
      ctx.fillStyle = style.ramp[idx]
      ctx.fillRect(x0, y0, x1 - x0, y1 - y0)
      col = end + 1
    }
  }
}

// =========================================================================================
// ENTRÉE 1 — points bruts (rejeu 2D) : accumulation de trajectoires + lissage gaussien.
// =========================================================================================

/** Les deux lectures proposées. `kills` = les morts, à la position des victimes. */
export type HeatmapMode = 'presence' | 'kills'

/** Portées de temps (V2, 2026-08-18) : `match` = tout le film (défaut) ; `live` = joué
 *  jusqu'à l'image courante, rien après. */
export type HeatmapSpan = 'match' | 'live'

/** Géométrie de lissage (m) : cellule visée (quart de σ), rayon d'un ENGAGEMENT, troncature
 *  du noyau (σ, 13,5 % perdus), cellules mini par σ (grandit sur BTB), plafond (200k=800ko). */
export const HEAT_CELL_M = 0.5
export const HEAT_SIGMA_M = 2
const KERNEL_SIGMAS = 2
const SIGMA_MIN_CELLS = 4
const HEAT_MAX_CELLS = 200_000

/** Trou d'échantillonnage (ms) au-delà duquel on ignore où le joueur était, plutôt que
 *  d'inventer une présence. */
const HEAT_MAX_GAP_MS = 1_000

/** Quantiles d'étalonnage (bas/haut) : un seul point extrême n'écrase donc pas la carte. */
const HEAT_Q_LOW = 0.5
const HEAT_Q_HIGH = 0.95

/** Repère PARTAGÉ d'une grille (côté de cellule en m monde, dimensions, coin monde de la
 *  cellule (0,0)) — commun aux deux entrées, évite de promener 5 nombres. */
export interface GridFrame {
  cell: number
  nx: number
  ny: number
  minX: number
  minY: number
}

/** HeatGrid — carte cuite depuis des points bruts : une grandeur dense (`Float32Array`)
 *  par cellule, et l'échelle qui la lit. `value` porte 0 là où personne n'est passé (VIDE,
 *  ne se peint pas). lo/hi = p50/p95 (au-delà, la couleur sature). */
export interface HeatGrid extends GridFrame {
  mode: HeatmapMode
  value: Float32Array
  lo: number
  hi: number
  /** Nombre de cellules fréquentées. 0 = rien à peindre. */
  filled: number
}

/** Une mort DATÉE : position + frame — la portée `live` ne peut pas compter une mort à venir. */
export interface HeatDeath extends XY {
  frame: number
}

/**
 * buildHeatmap cuit la carte, UNE fois : accumulation, lissage, étalonnage. Rend null quand
 * rien n'est mesurable. `deaths` sont les positions de MORT relues par `buildKillFx`, déjà
 * datées. `untilFrame` borne l'ACCUMULATION à l'image courante (portée `live`), jamais
 * l'étalonnage : la rampe se recalcule sur ce qui est mesuré à cet instant.
 */
export function buildHeatmap(
  doc: ReplayDocumentReady,
  bounds: ReplayBounds,
  mode: HeatmapMode,
  deaths: readonly HeatDeath[],
  untilFrame?: number,
): HeatGrid | null {
  const cell = cellSizeFor(bounds)
  const sigma = Math.max(HEAT_SIGMA_M, cell * SIGMA_MIN_CELLS)
  const radius = Math.max(1, Math.ceil((KERNEL_SIGMAS * sigma) / cell))
  // La grille DÉBORDE des bornes du rayon du noyau : sans cette marge, la masse déposée au
  // ras du bord serait rognée et le bord paraîtrait froid alors qu'il est joué.
  const g: GridFrame = {
    cell,
    nx: Math.ceil((bounds.maxX - bounds.minX) / cell) + 2 * radius + 1,
    ny: Math.ceil((bounds.maxY - bounds.minY) / cell) + 2 * radius + 1,
    minX: bounds.minX - radius * cell,
    minY: bounds.minY - radius * cell,
  }
  const raw = new Float32Array(g.nx * g.ny)
  const until = untilFrame === undefined ? Infinity : untilFrame
  const deposited =
    mode === 'kills'
      ? accumulateDeaths(raw, g, deaths, until)
      : accumulatePresence(raw, g, doc, until)
  if (deposited === 0) return null

  const value = blur(raw, g.nx, g.ny, gaussianKernel(sigma, cell, radius))
  const scale = scaleOf(value)
  if (scale.filled === 0) return null
  return { mode, ...g, value, lo: scale.lo, hi: scale.hi, filled: scale.filled }
}

/** cellSizeFor part de HEAT_CELL_M et ne grossit que si la carte dépasse le plafond. */
function cellSizeFor(bounds: ReplayBounds): number {
  const w = Math.max(bounds.maxX - bounds.minX, 1)
  const h = Math.max(bounds.maxY - bounds.minY, 1)
  return Math.max(HEAT_CELL_M, Math.sqrt((w * h) / HEAT_MAX_CELLS))
}

/**
 * accumulatePresence dépose, par position échantillonnée, le TEMPS qu'elle représente (demi
 * intervalle précédent + demi suivant, quadrature du trapèze). Une vie d'un seul point ne
 * dépose rien. Rend le nombre de dépôts.
 */
function accumulatePresence(
  raw: Float32Array,
  g: GridFrame,
  doc: ReplayDocumentReady,
  until: number,
): number {
  const maxGap = msToFrames(HEAT_MAX_GAP_MS, doc)
  let n = 0
  for (const track of doc.tracks) {
    const pts = track.points
    for (let i = 0; i < pts.length; i++) {
      // La borne est SUR LE POINT, pas sur la vie : une vie qui court encore dépose ce
      // qu'elle a parcouru jusqu'ici, et rien de son avenir.
      if (pts[i].t > until) break
      let frames = 0
      if (i > 0) frames += Math.min(pts[i].t - pts[i - 1].t, maxGap) / 2
      if (i + 1 < pts.length) frames += Math.min(pts[i + 1].t - pts[i].t, maxGap) / 2
      if (!(frames > 0)) continue
      if (deposit(raw, g, pts[i].x, pts[i].y, frameToMs(frames, doc) / 1000)) n++
    }
  }
  return n
}

/** accumulateDeaths dépose une mort par position de victime, jamais une qui n'a pas eu lieu. */
function accumulateDeaths(
  raw: Float32Array,
  g: GridFrame,
  deaths: readonly HeatDeath[],
  until: number,
): number {
  let n = 0
  for (const d of deaths) {
    if (d.frame > until) continue
    if (deposit(raw, g, d.x, d.y, 1)) n++
  }
  return n
}

/** deposit ajoute `w` à la cellule qui contient (x, y). Hors grille : rien, jamais un bord. */
function deposit(raw: Float32Array, g: GridFrame, x: number, y: number, w: number): boolean {
  const i = Math.floor((x - g.minX) / g.cell)
  const j = Math.floor((y - g.minY) / g.cell)
  if (i < 0 || i >= g.nx || j < 0 || j >= g.ny) return false
  raw[j * g.nx + i] += w
  return true
}

/** gaussianKernel rend un noyau 1D NORMALISÉ (somme 1) : le lissage conserve la grandeur. */
function gaussianKernel(sigma: number, cell: number, radius: number): Float64Array {
  const k = new Float64Array(2 * radius + 1)
  let sum = 0
  for (let d = -radius; d <= radius; d++) {
    const v = Math.exp(-((d * cell) ** 2) / (2 * sigma * sigma))
    k[d + radius] = v
    sum += v
  }
  for (let i = 0; i < k.length; i++) k[i] /= sum
  return k
}

/** blur applique le noyau en DEUX passes 1D (séparabilité du gaussien). Hors grille = 0,
 *  jamais un bord répété — la marge de `buildHeatmap` fait que ce bord ne porte rien de réel. */
function blur(src: Float32Array, nx: number, ny: number, kernel: Float64Array): Float32Array {
  const r = (kernel.length - 1) >> 1
  const tmp = new Float32Array(src.length)
  for (let j = 0; j < ny; j++) {
    const row = j * nx
    for (let i = 0; i < nx; i++) {
      let s = 0
      for (let d = -r; d <= r; d++) {
        const x = i + d
        if (x >= 0 && x < nx) s += src[row + x] * kernel[d + r]
      }
      tmp[row + i] = s
    }
  }
  const out = new Float32Array(src.length)
  for (let j = 0; j < ny; j++) {
    for (let i = 0; i < nx; i++) {
      let s = 0
      for (let d = -r; d <= r; d++) {
        const y = j + d
        if (y >= 0 && y < ny) s += tmp[y * nx + i] * kernel[d + r]
      }
      out[j * nx + i] = s
    }
  }
  return out
}

/** scaleOf étalonne sur les QUANTILES des cellules fréquentées (valeur > 0, sinon p50 tombe
 *  à zéro). Échelle dégénérée (toutes égales) : on retombe sur [0, max]. */
function scaleOf(value: Float32Array): { lo: number; hi: number; filled: number } {
  const pos: number[] = []
  let max = 0
  for (let i = 0; i < value.length; i++) {
    const v = value[i]
    if (v > 0) {
      pos.push(v)
      if (v > max) max = v
    }
  }
  if (pos.length === 0) return { lo: 0, hi: 0, filled: 0 }
  pos.sort((a, b) => a - b)
  const lo = quantile(pos, HEAT_Q_LOW)
  const hi = quantile(pos, HEAT_Q_HIGH)
  if (hi > lo) return { lo, hi, filled: pos.length }
  return { lo: 0, hi: max, filled: pos.length }
}

function quantile(sorted: number[], q: number): number {
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * q))]
}

/** heatIntensity rend la position d'une cellule sur la rampe, dans [0, 1] — ou null quand
 *  elle n'a jamais été atteinte (rien à peindre, pas « froid »). */
export function heatIntensity(grid: HeatGrid, index: number): number | null {
  return cellIntensity(grid.value[index], { lo: grid.lo, hi: grid.hi })
}

/** Cadrage déjà résolu pour `drawHeatmapLayer` (l'appelant projette via son `CanvasView` —
 *  ce noyau ignore ce type, pour ne jamais importer une feature depuis `lib/`). */
export interface HeatLayerView {
  /** Coin (minX, minY + ny·cell) du monde, déjà projeté en pixels canvas. */
  topLeft: XY
  /** Taille d'une cellule, en pixels canvas (`grid.cell * scaleOf(view)`). */
  step: number
}

/** drawHeatmapLayer peint une grille dense. LA LIGNE 0 EST LA PLUS BASSE EN MONDE, donc la
 *  plus BASSE au canvas (axe Y inversé, comme tout le rejeu) — seul ajout au noyau commun. */
export function drawHeatmapLayer(
  ctx: CanvasRenderingContext2D,
  grid: HeatGrid,
  view: HeatLayerView,
  style: HeatmapStyle,
): void {
  if (!(view.step > 0)) return
  const source: HeatSource = {
    nx: grid.nx,
    ny: grid.ny,
    filled: grid.filled,
    scale: { lo: grid.lo, hi: grid.hi },
    valueAt: (row, col) => grid.value[row * grid.nx + col],
  }
  const geometry: HeatGeometry = {
    xAt: (col) => view.topLeft.x + col * view.step,
    rowPixels: (row) => ({
      top: view.topLeft.y + (grid.ny - 1 - row) * view.step,
      bottom: view.topLeft.y + (grid.ny - row) * view.step,
    }),
  }
  drawHeatmap(ctx, source, geometry, style)
}

// =========================================================================================
// ENTRÉE 2 — cellules pré-agrégées (tactique) : le serveur a déjà accumulé et échelonné.
// =========================================================================================

/** Une cellule agrégée par le serveur : coordonnées grille et valeur. */
export interface TacticalCell {
  col: number
  row: number
  value: number
}

/** TacticalGrid — grille prête à peindre : cellules éparses + échelle (p50/p95 servis,
 *  pas recalculés). */
export interface TacticalGrid extends GridFrame {
  cells: TacticalCell[]
  lo: number
  hi: number
  /** Nombre de cellules fréquentées (non nulles). */
  filled: number
}

/** buildTacticalGrid assemble le format de dessin depuis les cellules serveur : l'échelle
 *  (p50, p95) est déjà fournie, rien n'est recalculé. */
export function buildTacticalGrid(
  cells: TacticalCell[],
  gridSize: GridFrame,
  scale: { lo: number; hi: number },
  filled: number,
): TacticalGrid {
  return { ...gridSize, cells, lo: scale.lo, hi: scale.hi, filled }
}

/** tacticalIntensity : même règle que `heatIntensity`, pour une grille éparse (valeur passée
 *  directement, pas d'index). */
export function tacticalIntensity(grid: TacticalGrid, value: number): number | null {
  return cellIntensity(value, { lo: grid.lo, hi: grid.hi })
}

/** Cadrage pour `drawTacticalHeatmap` : canvas cadré EXACTEMENT sur les bornes du raster
 *  (comme `TacticalPlanCard` pour le fond) — donc PAS de Y-flip, la ligne 0 est en haut. */
export interface TacticalLayerView {
  /** Origine canvas du coin (min_x, min_y) du monde — généralement (0, 0). */
  topLeftWorld: XY
  /** Pixels canvas par mètre monde. */
  scale: number
}

/** drawTacticalHeatmap peint une grille éparse. PAS DE Y-FLIP (contraire au rejeu) — seul
 *  ajout au noyau commun. */
export function drawTacticalHeatmap(
  ctx: CanvasRenderingContext2D,
  grid: TacticalGrid,
  view: TacticalLayerView,
  style: HeatmapStyle,
): void {
  const cellPixels = grid.cell * view.scale
  const byPosition = new Map<string, number>()
  for (const cell of grid.cells) byPosition.set(`${cell.col},${cell.row}`, cell.value)
  const source: HeatSource = {
    nx: grid.nx,
    ny: grid.ny,
    filled: grid.filled,
    scale: { lo: grid.lo, hi: grid.hi },
    valueAt: (row, col) => byPosition.get(`${col},${row}`) ?? 0,
  }
  const geometry: HeatGeometry = {
    xAt: (col) => view.topLeftWorld.x + col * cellPixels,
    rowPixels: (row) => ({
      top: view.topLeftWorld.y + row * cellPixels,
      bottom: view.topLeftWorld.y + (row + 1) * cellPixels,
    }),
  }
  drawHeatmap(ctx, source, geometry, style)
}
