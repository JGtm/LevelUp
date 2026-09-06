/**
 * heatmapLayer.ts — LA CARTE DE CHALEUR du rejeu : où le match s'est vraiment joué.
 *
 * DEUX GRANDEURS, JAMAIS MÉLANGÉES. La PRÉSENCE est un TEMPS (secondes passées par les
 * joueurs sur une cellule) ; les ÉLIMINATIONS sont un COMPTE, déposé à la position de la
 * VICTIME — c'est l'endroit où l'on meurt qui dit le danger d'un lieu, pas celui d'où l'on
 * tire. Le « froid » n'est pas un troisième calcul : c'est le bas de la même échelle.
 *
 * CE QUI N'EST PAS PEINT COMPTE AUTANT QUE CE QUI L'EST. Une cellule jamais atteinte reste
 * VIDE, elle n'est pas peinte en froid : « froid » veut dire peu fréquenté, « rien » veut
 * dire jamais vu. Les confondre ferait dire à la carte qu'un mur est un endroit calme.
 *
 * LA GRILLE EST EN MÈTRES, PAS EN PIXELS DU FOND. Le plan visait « la grille du raster de
 * fond » ; vérifié sur pièces, ce raster vaut 0,092 m/px (sidecar Ridgeline : 1633 px pour
 * ~150 m), soit près de deux millions de cellules pour une seule carte — cent fois plus fin
 * que le lissage, et il n'existe que pour les 21 cartes qui ont une image. La grille suit
 * donc le patron du sol reconstruit (mapFloor.ts) : un pas en MÈTRES sur les bornes de la
 * scène, donc la même projection `worldToCanvas` que tout le reste du canvas — le calage
 * est exact là où il compte, et la carte de chaleur existe sur TOUTES les cartes.
 *
 * LE LISSAGE EST UN RAYON DU MONDE (mètres), pas un flou d'écran : un point chaud de 2 m
 * reste un point chaud de 2 m quelle que soit la taille de la fenêtre. Noyau gaussien
 * séparable, tronqué à 2 σ (13,5 % du sommet, ~5 % de la masse perdue) — au-delà, la
 * traînée du noyau peindrait un halo plus large que le lieu qu'elle décrit.
 *
 * L'ÉCHELLE EST QUANTILE (p50 -> p95 des cellules fréquentées), jamais le maximum : un seul
 * point extrême — un joueur mort qui reste 40 s au même endroit — écraserait tout le reste
 * au bas de la rampe. Au-delà de p95 la couleur SATURE : c'est assumé, c'est ce qui rend
 * les zones chaudes comparables entre elles.
 *
 * LA RAMPE A TROIS POINTS depuis le 2026-08-18 (A8) : bleu -> rouge -> violet. Le violet ne
 * peint que le HAUT de l'échelle, là où la saturation commence — 5 % des cellules du témoin.
 * C'est ce que « aux extrêmes rares » veut dire, et c'est pour cela que la couleur change
 * deux fois plutôt qu'une : une rampe mono-teinte dit « plus ou moins », elle ne dit jamais
 * « et là, beaucoup plus ». Les trois points sont des TOKENS (heatmapRampTokens('intensity')),
 * résolus par l'appelant : ce fichier ne nomme toujours aucune couleur.
 *
 * Pas de React : logique pure + un CanvasRenderingContext2D (même règle que replayDraw).
 */
import { hexToRgba } from '@/components/charts/_utils'
import type { ReplayBounds } from '@/lib/api/types'

import { frameToMs, msToFrames, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { type CanvasView, projectTo, scaleOf as viewScale } from '../model/replayView'

/** Les deux lectures proposées. `kills` = les morts, à la position des victimes. */
export type HeatmapMode = 'presence' | 'kills'

/**
 * LES DEUX PORTÉES DE TEMPS de la carte de chaleur (V2, retour utilisateur du 2026-08-18).
 *
 * CE QUE LA MESURE A ÉTABLI AVANT DE CODER : la carte de chaleur du 16/08 est DÉJÀ celle de
 * TOUTE LA PARTIE — `accumulatePresence` parcourt tous les points de toutes les vies, et le
 * calque est cuit UNE fois hors écran. Il n'existait donc aucun mode « au fur et à mesure »,
 * contrairement à ce que la lecture à l'écran laissait croire. Ce qui manquait, c'est la
 * SECONDE portée ; c'est elle que ce type introduit, et `match` reste le défaut.
 *
 *  - `match` : tout le film, du début à la fin. La lecture d'analyse d'après-match — « en un
 *    bouton on voit la heatmap de toute la partie », demande du 18/08 — et le comportement
 *    inchangé depuis le 16/08 ;
 *  - `live`  : ce qui a été joué JUSQU'À L'IMAGE COURANTE, et rien après. La carte se remplit
 *    en même temps que le rejeu, ce qui répond à « la heatmap au fur et à mesure ».
 */
export type HeatmapSpan = 'match' | 'live'

/** Côté de cellule visé, en mètres : le quart d'un σ de lissage — assez fin pour que le
 *  noyau ne se voie pas en escalier, assez gros pour qu'une carte tienne en mémoire. */
export const HEAT_CELL_M = 0.5

/** Rayon de lissage, en mètres. C'est l'échelle d'un ENGAGEMENT (un duel, une rampe, une
 *  entrée de couloir) : à 1 m la carte se couvre de mouchetures individuelles, à 4 m une
 *  arène entière se fond en une seule tache. */
export const HEAT_SIGMA_M = 2

/** Troncature du noyau, en σ. */
const KERNEL_SIGMAS = 2

/** Cellules minimales par σ : si la carte force une cellule plus grosse (BTB), σ grandit
 *  avec elle — sans quoi le noyau serait échantillonné trop grossièrement pour lisser. */
const SIGMA_MIN_CELLS = 4

/** Plafond de cellules : au-delà, la cellule grossit. Une grille de 200 000 cellules tient
 *  en 800 ko et se lisse en quelques dizaines de millisecondes, une fois par document. */
const HEAT_MAX_CELLS = 200_000

/**
 * Trou d'échantillonnage au-delà duquel une durée n'est plus attribuable, en ms. Les
 * trajectoires sont échantillonnées à 100 ms ; passé une seconde sans mesure, on ignore où
 * le joueur était — attribuer tout l'intervalle à ses deux extrémités inventerait une
 * présence. On garde la seconde mesurable et on laisse tomber le reste.
 */
const HEAT_MAX_GAP_MS = 1_000

/** Quantiles d'étalonnage : bas et haut de la rampe (décision 5 du plan). */
const HEAT_Q_LOW = 0.5
const HEAT_Q_HIGH = 0.95

/**
 * Opacités des extrémités. Le bas reste franchement visible sans effacer le décor — c'est la
 * moitié froide de l'échelle, et l'utilisateur a demandé à voir AUSSI les lieux froids.
 * L'opacité monte avec l'intensité, comme la couleur : sur un fond chargé, une opacité
 * constante noierait les neuf dixièmes de la carte pour montrer un dixième.
 *
 * LE PLAFOND PASSE DE 0,55 À 0,75 LE 2026-08-18 (A8 : « c'est pas mal mais à accentuer un
 * peu »). Deux leviers étaient possibles et le lot R2-V les a CHIFFRÉS avant qu'on choisisse :
 * abaisser le quantile bas déplace le plancher et rien d'autre (+1,4 pt de cellules au-dessus
 * d'alpha 0,30), relever le plafond éclaire tout le haut de l'échelle (+7,1 pt). Le plafond
 * fait cinq fois plus, et c'est lui qui a été retenu — l'étalonnage quantile (p50 -> p95) ne
 * bouge donc PAS.
 */
const HEAT_ALPHA_MIN = 0.12
const HEAT_ALPHA_MAX = 0.75

/** Paliers de la rampe précalculée : un `rgba()` par palier, indexé pendant le dessin. */
export const HEAT_RAMP_STEPS = 64

/**
 * HeatGrid — la carte de chaleur cuite : une grandeur par cellule, et l'échelle qui la lit.
 * `value` porte 0 là où personne n'est passé : c'est le VIDE, il ne se peint pas.
 */
export interface HeatGrid {
  mode: HeatmapMode
  /** Côté d'une cellule, en mètres monde. */
  cell: number
  nx: number
  ny: number
  /** Coin monde (x minimal, y minimal) de la cellule (0,0). */
  minX: number
  minY: number
  /** Valeur lissée : des SECONDES en présence, un NOMBRE de morts en éliminations. */
  value: Float32Array
  /** Bas d'échelle (p50 des cellules fréquentées). */
  lo: number
  /** Haut d'échelle (p95) — au-delà, la couleur sature. */
  hi: number
  /** Nombre de cellules fréquentées. 0 = rien à peindre. */
  filled: number
}

/** Repère de la grille, sans ses valeurs (évite de promener 5 nombres). */
interface GridFrame {
  cell: number
  nx: number
  ny: number
  minX: number
  minY: number
}

/**
 * Une mort DATÉE : sa position, et la frame où elle a eu lieu. La date n'existe que pour la
 * portée `live` — sans elle, une carte « jusqu'à l'image courante » compterait des morts qui
 * ne se sont pas encore produites.
 */
export interface HeatDeath extends XY {
  frame: number
}

/**
 * buildHeatmap cuit la carte de chaleur d'un document, UNE fois (patron `buildShotFx`) :
 * accumulation, lissage, étalonnage. Rend null quand rien n'est mesurable — pas de calque
 * plutôt qu'un calque vide.
 *
 * `deaths` sont les positions de MORT relues par `buildKillFx` (champ `deathX`/`deathY`),
 * datées : la carte des éliminations ne relit pas les trajectoires une seconde fois.
 *
 * `untilFrame` BORNE LA MESURE À L'IMAGE COURANTE (portée `live`). Absent = toute la partie,
 * le comportement du 16/08. La borne s'applique à L'ACCUMULATION, jamais à l'étalonnage : la
 * rampe se recalcule donc sur ce qui est mesuré à cet instant, et une carte qui se remplit
 * reste lisible du début à la fin plutôt que de rester pâle pendant deux minutes.
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
 * accumulatePresence dépose, sur chaque position échantillonnée, le TEMPS qu'elle
 * représente : la moitié de l'intervalle qui la précède plus la moitié de celui qui la
 * suit (quadrature du trapèze). Une vie d'un seul point ne dépose rien — on ne mesure pas
 * une durée avec un seul instant. Rend le nombre de dépôts.
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

/** gaussianKernel rend un noyau 1D NORMALISÉ (somme 1) : le lissage conserve la grandeur —
 *  une cellule reste des secondes, la carte ne change pas d'unité en se lissant. */
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

/**
 * blur applique le noyau en DEUX passes 1D (séparabilité du gaussien) : le coût passe de
 * (2r+1)² à 2(2r+1) multiplications par cellule. Hors grille = 0, jamais un bord répété —
 * un bord répété entasserait de la masse au ras du cadre et y inventerait un point chaud
 * (la marge de `buildHeatmap` fait que ce bord ne porte rien de réel).
 */
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

/**
 * scaleOf étalonne la rampe sur les QUANTILES des cellules fréquentées (valeur > 0). Les
 * cellules vides sont exclues : sur une carte, l'immense majorité n'est jamais atteinte —
 * les compter mettrait p50 à zéro et l'échelle serait pilotée par le fond, pas par le jeu.
 *
 * Échelle dégénérée (toutes les cellules à la même valeur) : on retombe sur [0, max], soit
 * une carte uniformément chaude — ce qu'elle est.
 */
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

/**
 * heatIntensity rend la position d'une cellule sur la rampe, dans [0, 1] — ou null quand
 * la cellule n'a jamais été atteinte (rien à peindre, pas « froid »).
 */
export function heatIntensity(grid: HeatGrid, index: number): number | null {
  const v = grid.value[index]
  if (!(v > 0)) return null
  const span = grid.hi - grid.lo
  if (!(span > 0)) return 1
  const t = (v - grid.lo) / span
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
 * N POINTS, PAS DEUX (2026-08-18) : les arrêts arrivent DÉJÀ RÉSOLUS de l'appelant, dans
 * l'ordre bas -> haut (règle color-tokens : ce fichier ne nomme aucune couleur). Deux arrêts
 * donnent la rampe d'avant ; trois donnent le bleu -> rouge -> violet demandé. Ils sont
 * répartis UNIFORMÉMENT sur l'échelle : la position du point milieu est un choix d'écran, et
 * la placer ailleurs sans mesure reviendrait à inventer un seuil.
 *
 * Un arrêt illisible, ou moins de deux, rend une rampe VIDE — le calque ne se peint pas,
 * plutôt que de peindre une couleur inventée (même règle que `readInk`).
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

/** Cadrage du canvas (les mêmes paramètres que worldToCanvas, forme de replayDraw). */
/** Style du calque : rampe DÉJÀ résolue, et le rapport de pixels de l'écran. */
export interface HeatmapStyle {
  ramp: readonly string[]
  /** devicePixelRatio : les bords de cellule s'y alignent (cf. drawHeatmapLayer). */
  k: number
}

/**
 * drawHeatmapLayer peint la grille. Calque STATIQUE : l'appelant le cuit hors écran et le
 * recopie, comme le sol et les zones — la carte de chaleur porte tout le match, elle ne
 * dépend pas de l'image courante (donc `prefers-reduced-motion` est respectée par
 * construction : rien ne s'anime de soi-même).
 *
 * DEUX PRÉCAUTIONS CONTRE LE QUADRILLAGE PARASITE, le défaut documenté du sol reconstruit :
 * les cellules voisines de même palier sont peintes d'un SEUL rectangle (plages, comme
 * `floorRun`), et chaque bord est aligné sur un pixel PHYSIQUE de l'écran — sans quoi deux
 * rectangles translucides se partagent un pixel anti-crénelé et tracent une couture claire.
 */
export function drawHeatmapLayer(
  ctx: CanvasRenderingContext2D,
  grid: HeatGrid,
  view: CanvasView,
  style: HeatmapStyle,
): void {
  const last = style.ramp.length - 1
  if (grid.filled === 0 || last < 0) return
  const step = grid.cell * viewScale(view)
  if (!(step > 0)) return
  const topLeft = projectTo(view, { x: grid.minX, y: grid.minY + grid.ny * grid.cell })
  const snap = (v: number) => Math.round(v * style.k) / style.k
  for (let j = 0; j < grid.ny; j++) {
    const row = j * grid.nx
    // La ligne 0 de la grille est la plus BASSE en monde, donc la plus BASSE au canvas.
    const yTop = snap(topLeft.y + (grid.ny - 1 - j) * step)
    const yBottom = snap(topLeft.y + (grid.ny - j) * step)
    let i = 0
    while (i < grid.nx) {
      const idx = rampIndex(grid, row + i, last)
      if (idx < 0) {
        i++
        continue
      }
      let end = i
      while (end + 1 < grid.nx && rampIndex(grid, row + end + 1, last) === idx) end++
      const x0 = snap(topLeft.x + i * step)
      const x1 = snap(topLeft.x + (end + 1) * step)
      ctx.fillStyle = style.ramp[idx]
      ctx.fillRect(x0, yTop, x1 - x0, yBottom - yTop)
      i = end + 1
    }
  }
}

/** rampIndex rend le palier de rampe d'une cellule, ou -1 si elle n'a jamais été atteinte. */
function rampIndex(grid: HeatGrid, index: number, last: number): number {
  const t = heatIntensity(grid, index)
  return t === null ? -1 : Math.round(t * last)
}
