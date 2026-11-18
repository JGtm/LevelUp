/**
 * relief/engine — LE MOTEUR DU RELIEF : l'état d'UNE carte et ses calculs, sans worker ni DOM.
 *
 * C'est ce que le worker exécute (`workerHandler.ts`) et ce que le repli synchrone exécute à sa
 * place (`client.ts`) : le même code, donc le même résultat. Le moteur garde les calculs chers entre
 * deux demandes (conception §6.2) :
 *
 *   - l'ombrage, une fois par ORIENTATION (46 à 93 ms) ;
 *   - la pile des sols, à la PREMIERE coupe seulement (18 à 51 ms) ;
 *   - les coupes, au plus quatre en cache, par orientation et hauteur arrondie au quart d'unité
 *     (24 à 61 ms chacune) ;
 *   - la table de teinte, par couleur de base : un changement de palette redemande la TEINTE, pas la coupe.
 *
 * Rien de ce que `render` rend n'est partagé avec le cache : les tableaux d'une `ReliefFrame` sont
 * des copies, que l'appelant peut transférer sans abîmer le moteur.
 */
import { CUT_BAND_U } from './constants'
import { hasNoMesh } from './decodeData'
import { ReliefError } from './errors'
import { quantizeCut } from './floors'
import { buildFloorStack, type FloorStack } from './floorStack'
import { meshWalls, wallsInBand, type MeshWalls } from './floorMesh'
import { emptyLineSet } from './lineSet'
import { filterLinesByKeep } from './lineFilter'
import { shadeRelief } from './shading'
import { computeSlice, type ReliefSlice } from './slice'
import { buildTintTable, reliefRamp, tintRelief, type ReliefBaseColor } from './tint'
import type { QuarterTurns } from '../orientation'
import type { LineSet, ReliefData, ReliefHeights } from './types'

/** Ce qu'une carte apporte au moteur. */
export interface ReliefLoadInput {
  image: ReliefHeights
  data: ReliefData
}

/** Ce que le moteur dit d'une carte chargée. */
export interface ReliefInfo {
  width: number
  height: number
  /** Faux sans maillage : pas de coupe, « Tout » seulement. */
  hasMesh: boolean
}

/** Une demande de rendu. */
export interface ReliefRenderRequest {
  /** Quarts de tour horaires de la carte : choisit la direction de la lumière. */
  turns: QuarterTurns
  /** Hauteur de coupe en unités ; `null` pour « Tout ». Arrondie au quart d'unité. */
  cut: number | null
  /** Base de la teinte ; `null` quand seul le masque ou les traits sont demandés (le fond affiché est Plan ou Rendu). */
  base: ReliefBaseColor | null
  /** Demander les traits filtrés par la coupe. */
  lines: boolean
  /** Demander les masques de la coupe pour les fonds Rendu et Plan. */
  masks: boolean
}

/** Les traits d'une demande : ruptures et courbes filtrées par la coupe, murs du maillage dans la bande. */
export interface ReliefFrameLines {
  steps: LineSet
  contours: LineSet
  /** Vide sans coupe. */
  walls: LineSet
}

/** Le résultat d'un rendu. Les tableaux sont à l'appelant. */
export interface ReliefFrame {
  width: number
  height: number
  /** La hauteur de coupe appliquée (arrondie), `null` pour « Tout ». */
  cut: number | null
  /** Le fond Relief teinté, RGBA ; `null` si `base` n'était pas demandée. */
  image: Uint8ClampedArray | null
  /** Le masque d'opacité du fond Rendu ; `null` sans coupe ou sans `masks`. */
  alpha: Uint8Array | null
  /** Le masque d'opacité du fond Plan ; `null` sans coupe ou sans `masks`. */
  planAlpha: Uint8Array | null
  lines: ReliefFrameLines | null
}

/** Le nombre de coupes gardées en cache. */
const SLICE_CACHE_SIZE = 4

/** Un moteur pour une carte. */
export class ReliefEngine {
  private input: ReliefLoadInput | null = null
  private mesh = false
  private greys = new Map<QuarterTurns, Uint8Array>()
  private slices: Array<{ key: string; slice: ReliefSlice }> = []
  private stack: FloorStack | null = null
  private walls: MeshWalls | null = null
  private tints = new Map<string, Uint8ClampedArray>()

  /** load pose la carte (et jette l'état de la précédente). Refuse une image qui n'est pas celle du calage. */
  load(input: ReliefLoadInput): ReliefInfo {
    const { image, data } = input
    if (image.width !== data.calibration.widthPx || image.height !== data.calibration.heightPx) {
      throw new ReliefError('image-size', 'image et calage de tailles différentes')
    }
    this.dispose()
    this.input = input
    this.mesh = !hasNoMesh(data)
    return { width: image.width, height: image.height, hasMesh: this.mesh }
  }

  /** Jette tout l'état gardé. */
  dispose(): void {
    this.input = null
    this.greys.clear()
    this.slices = []
    this.stack = null
    this.walls = null
    this.tints.clear()
  }

  /** render calcule ce que la demande veut ; lève `ReliefError` si rien n'est chargé ou si la base est illisible. */
  render(req: ReliefRenderRequest): ReliefFrame {
    const { image, data } = this.loaded()
    const cut = req.cut === null || !this.mesh ? null : quantizeCut(req.cut)
    const table = req.base === null ? null : this.tintTable(req.base)
    const slice = this.slice(req.turns, cut)
    const tinted = table
      ? tintRelief({ width: image.width, height: image.height, grey: slice.grey, alpha: slice.alpha, flags: image.flags }, table)
      : null
    return {
      width: image.width,
      height: image.height,
      cut,
      image: tinted ? tinted.rgba : null,
      alpha: req.masks && cut !== null ? slice.alpha.slice() : null,
      planAlpha: req.masks && cut !== null && slice.planAlpha ? slice.planAlpha.slice() : null,
      lines: req.lines ? this.framedLines(slice, cut, data) : null,
    }
  }

  private loaded(): ReliefLoadInput {
    if (this.input === null) throw new ReliefError('not-loaded', 'aucune carte chargée')
    return this.input
  }

  private tintTable(base: ReliefBaseColor): Uint8ClampedArray {
    const key = typeof base === 'string' ? base : base.join(',')
    const known = this.tints.get(key)
    if (known) return known
    const ramp = reliefRamp(base)
    if (!ramp) throw new ReliefError('tint-invalid', `couleur de base invalide : ${key}`)
    const table = buildTintTable(ramp)
    this.tints.set(key, table)
    return table
  }

  private grey(turns: QuarterTurns): Uint8Array {
    const { image, data } = this.loaded()
    const known = this.greys.get(turns)
    if (known) return known
    const grey = shadeRelief(image, data.height, data.calibration.metersPerPixel, turns)
    this.greys.set(turns, grey)
    return grey
  }

  private floorStack(): FloorStack {
    if (this.stack === null) this.stack = buildFloorStack(this.loaded().data)
    return this.stack
  }

  /** La coupe de cette orientation et de cette hauteur : en cache, ou calculée (les quatre dernières restent). */
  private slice(turns: QuarterTurns, cut: number | null): ReliefSlice {
    const key = `${turns}|${cut}`
    const hit = this.slices.find((s) => s.key === key)
    if (hit) return hit.slice
    const { image, data } = this.loaded()
    const slice = computeSlice(
      {
        image,
        grey: this.grey(turns),
        stack: cut === null ? null : this.floorStack(),
        mpp: data.calibration.metersPerPixel,
        ramp: data.height,
        tolerance: data.floors.tolerance,
      },
      cut,
      { plan: true },
    )
    this.slices.unshift({ key, slice })
    if (this.slices.length > SLICE_CACHE_SIZE) this.slices.length = SLICE_CACHE_SIZE
    return slice
  }

  /** Les traits : sans coupe, ceux du fichier ; avec coupe, filtrés par la garde, plus les murs de la bande. */
  private framedLines(slice: ReliefSlice, cut: number | null, data: ReliefData): ReliefFrameLines {
    const { image } = this.loaded()
    if (cut === null) {
      return { steps: cloneLines(data.lines.steps), contours: cloneLines(data.lines.contours), walls: emptyLineSet() }
    }
    const grid = { width: image.width, height: image.height, keep: slice.keep }
    const mpp = data.calibration.metersPerPixel
    if (this.walls === null) this.walls = meshWalls(data.floors.mesh, data.calibration)
    return {
      steps: filterLinesByKeep(data.lines.steps, grid, mpp, true),
      contours: filterLinesByKeep(data.lines.contours, grid, mpp, false),
      walls: wallsInBand(this.walls, cut, CUT_BAND_U),
    }
  }
}

function cloneLines(lines: LineSet): LineSet {
  return { points: lines.points.slice(), offsets: lines.offsets.slice() }
}
