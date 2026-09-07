/**
 * heatPaint.ts — Peinture de chaleur pour l'analyse tactique (Phase 5).
 *
 * Copie de heatmapLayer.ts (2026-09-07) ; fusion avec lib/replay/ après lot D — cf. DECOUVERTES.
 *
 * DIFFÉRENCE CLÉS PAR RAPPORT À heatmapLayer.ts :
 * - Le serveur a DÉJÀ rasterisé et agrégé les cellules (col, row, valeur)
 * - Pas d'accumulation de points bruts, pas de trajectoires à lisser
 * - Pas de `ReplayDocumentReady` ni de morts datées — juste une grille pré-lissée
 * - L'échelle est SERVIE par le serveur (p50, p95)
 *
 * CE QUI EST REPRIS SANS CHANGEMENT :
 * - Rampe de chaleur : 3 points de couleur, quantiles p50 -> p95, opacités montantes
 * - Dessin canvas : cellules alignées pixels, plages de couleur uniforme
 * - Palette : tokens sémantiques résolus, aucune couleur en dur
 *
 * Pas de React : logique pure + CanvasRenderingContext2D.
 */
import { hexToRgba } from '@/components/charts/_utils'

/** Une cellule agrégée par le serveur : coordonnées grille et valeur. */
export interface TacticalCell {
  col: number
  row: number
  value: number
}

/**
 * TacticalGrid — la grille de chaleur prête à peindre : cellules + échelle.
 * Le serveur a déjà fait l'accumulation et le lissage.
 */
export interface TacticalGrid {
  /** Pas d'une cellule, en mètres monde. */
  cell: number
  /** Nombre de colonnes. */
  nx: number
  /** Nombre de lignes. */
  ny: number
  /** Coin monde (x minimal, y minimal) de la cellule (0,0). */
  minX: number
  minY: number
  /** Cellules agrégées (col, row, value). */
  cells: TacticalCell[]
  /** Bas d'échelle (p50 des cellules fréquentées). */
  lo: number
  /** Haut d'échelle (p95) — au-delà, la couleur sature. */
  hi: number
  /** Nombre de cellules fréquentées (non-zéro). */
  filled: number
}

/**
 * Paliers de la rampe précalculée : un `rgba()` par palier, indexé pendant le dessin.
 * Même valeur que heatmapLayer.ts : 64 couleurs.
 */
export const HEAT_RAMP_STEPS = 64

/**
 * Opacités : mêmes valeurs que heatmapLayer.ts (décision 2026-08-18, A8).
 */
const HEAT_ALPHA_MIN = 0.12
const HEAT_ALPHA_MAX = 0.75

/**
 * buildTacticalGrid construit la grille de peinture à partir des cellules
 * serveur. Le serveur a déjà fourni l'échelle (p50, p95) ; on enrichit juste
 * le format pour le dessin.
 */
export function buildTacticalGrid(
  cells: TacticalCell[],
  gridSize: { cell: number; nx: number; ny: number; minX: number; minY: number },
  scale: { lo: number; hi: number },
  filled: number,
): TacticalGrid {
  return {
    cell: gridSize.cell,
    nx: gridSize.nx,
    ny: gridSize.ny,
    minX: gridSize.minX,
    minY: gridSize.minY,
    cells,
    lo: scale.lo,
    hi: scale.hi,
    filled,
  }
}

/**
 * tacticalIntensity rend la position d'une cellule sur la rampe, dans [0, 1] — ou null
 * si la cellule n'a jamais été atteinte.
 *
 * Identique à heatIntensity, adaptée au nom du domaine.
 */
export function tacticalIntensity(grid: TacticalGrid, value: number): number | null {
  if (!(value > 0)) return null
  const span = grid.hi - grid.lo
  if (!(span > 0)) return 1
  const t = (value - grid.lo) / span
  return t < 0 ? 0 : t > 1 ? 1 : t
}

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
 * heatRamp précalcule la rampe en `rgba()` : couleur ET opacité montent ensemble.
 *
 * Identique à heatmapLayer.ts : N points (2+ requis), répartis uniformément sur l'échelle.
 * Un arrêt illisible ou < 2 points rend une rampe VIDE.
 */
export function heatRamp(stops: readonly string[]): string[] {
  const rgb = stops.map(parseHex)
  if (rgb.length < 2 || rgb.some((c) => c === null)) return []
  const points = rgb as Rgb[]
  const segments = points.length - 1
  const out: string[] = []
  for (let i = 0; i < HEAT_RAMP_STEPS; i++) {
    const t = i / (HEAT_RAMP_STEPS - 1)
    const seg = Math.min(segments - 1, Math.floor(t * segments))
    const u = t * segments - seg
    const alpha = HEAT_ALPHA_MIN + (HEAT_ALPHA_MAX - HEAT_ALPHA_MIN) * t
    out.push(hexToRgba(mixHex(points[seg], points[seg + 1], u), Number(alpha.toFixed(3))))
  }
  return out
}

/** Cadrage du canvas. */
interface CanvasView {
  width: number
  height: number
  /** Coin monde du coin haut-gauche du canvas. */
  topLeftWorld: { x: number; y: number }
  /** Pixels par mètre monde. */
  scale: number
  /** devicePixelRatio : les bords de cellule s'y alignent. */
  k: number
}

/** Style du calque : rampe DÉJÀ résolue, et le rapport de pixels de l'écran. */
export interface TacticalPaintStyle {
  ramp: readonly string[]
  k: number
}

/**
 * drawTacticalHeatmap peint la grille de chaleur sur le canvas.
 *
 * STRATÉGIE : cellules voisines de même palier sont peintes d'un SEUL rectangle (plages),
 * et chaque bord est aligné sur un pixel PHYSIQUE — sans quoi deux rectangles translucides
 * se partagent un pixel anti-crénelé et tracent une couture (mêmes précautions que
 * drawHeatmapLayer, champ documenté).
 */
export function drawTacticalHeatmap(
  ctx: CanvasRenderingContext2D,
  grid: TacticalGrid,
  view: CanvasView,
  style: TacticalPaintStyle,
): void {
  const last = style.ramp.length - 1
  if (grid.filled === 0 || last < 0) return

  const snap = (v: number) => Math.round(v * style.k) / style.k

  // Index les cellules par (col, row) pour accès O(1).
  const cellMap = new Map<string, TacticalCell>()
  for (const cell of grid.cells) {
    cellMap.set(`${cell.col},${cell.row}`, cell)
  }

  const cellPixels = grid.cell * view.scale

  // Peint ligne par ligne, en fusionnant les plages de même couleur.
  for (let row = 0; row < grid.ny; row++) {
    let col = 0
    while (col < grid.nx) {
      const key = `${col},${row}`
      const cell = cellMap.get(key)
      const idx = cell ? rampIndexForValue(grid, cell.value, last) : -1

      if (idx < 0) {
        col++
        continue
      }

      // Fusionner avec les cellules voisines du même palier.
      let endCol = col
      while (endCol + 1 < grid.nx) {
        const nextKey = `${endCol + 1},${row}`
        const nextCell = cellMap.get(nextKey)
        const nextIdx = nextCell ? rampIndexForValue(grid, nextCell.value, last) : -1
        if (nextIdx !== idx) break
        endCol++
      }

      // Monde vers canvas.
      const x0 = snap(view.topLeftWorld.x + col * cellPixels)
      const x1 = snap(view.topLeftWorld.x + (endCol + 1) * cellPixels)
      const y0 = snap(view.topLeftWorld.y + row * cellPixels)
      const y1 = snap(view.topLeftWorld.y + (row + 1) * cellPixels)

      ctx.fillStyle = style.ramp[idx]
      ctx.fillRect(x0, y0, x1 - x0, y1 - y0)

      col = endCol + 1
    }
  }
}

/** rampIndexForValue rend le palier de rampe d'une valeur, ou -1 si zéro. */
function rampIndexForValue(grid: TacticalGrid, value: number, last: number): number {
  const t = tacticalIntensity(grid, value)
  return t === null ? -1 : Math.round(t * last)
}
