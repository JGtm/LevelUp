/**
 * relief/decodeData — LE JSON DU RELIEF : validé et décodé en structures typées.
 *
 * Le fichier (`<clé>.relief.json`, conception §1.2) porte le calage, les bornes de la rampe, les
 * étages, les sols du maillage (sommets indexés, faces convexes) et les traits. Ce module est le
 * SEUL lecteur du transport : il refuse ce qui n'est pas un relief (erreur typée, l'appelant
 * retombe sur le Plan) et rend des tableaux typés prêts à l'emploi.
 *
 * TRAITS (§1.2, §3.5) : une polyligne est `[x0, y0, dx1, dy1, …]` en DIXIEMES de pixel entiers ;
 * le premier couple est absolu, les suivants sont l'écart au point précédent. Le décodage rend des
 * sommets en pixels (`u / 10`), le centre du pixel étant l'entier.
 *
 * SOMMETS : centièmes d'unité entiers, monde, `x, y, z` à plat ; rendus en unités.
 *
 * Les clés inconnues sont ignorées (le fichier d'un serveur plus récent reste lisible tant que
 * `schemaVersion` est celle que ce module connaît) ; `floors.merge` et `floors.tolerance`, absents
 * d'un fichier plus ancien, valent leurs défauts de `constants.ts`.
 */
import { DEFAULT_FLOOR_MERGE_U, DEFAULT_FLOOR_TOLERANCE_U } from './constants'
import { ReliefError } from './errors'
import { LineSetBuilder } from './lineSet'
import type {
  FloorMesh,
  LineSet,
  ReliefCalibration,
  ReliefData,
  ReliefDegradation,
  ReliefFloors,
  ReliefHeightMeta,
  ReliefLines,
} from './types'

/** La version de schéma que ce module sait lire. */
export const RELIEF_SCHEMA_VERSION = 1

/** Les codes de dégradation connus (liste fermée côté serveur). */
const KNOWN_DEGRADATIONS: readonly ReliefDegradation[] = ['sans-maillage', 'hauteurs-ecretees', 'pile-tronquee']

/** Dixièmes de pixel par pixel : l'unité des coordonnées de traits du fichier. */
const TENTHS_PER_PIXEL = 10

/** Centièmes d'unité par unité : l'unité des sommets du fichier. */
const CENTI_PER_UNIT = 100

type Obj = Record<string, unknown>

function invalid(what: string): never {
  throw new ReliefError('data-invalid', `relief.json : ${what}`)
}

function asObject(v: unknown, what: string): Obj {
  if (typeof v !== 'object' || v === null || Array.isArray(v)) invalid(`${what} n'est pas un objet`)
  return v as Obj
}

function asNumber(v: unknown, what: string): number {
  if (typeof v !== 'number' || !Number.isFinite(v)) invalid(`${what} n'est pas un nombre fini`)
  return v
}

function asArray(v: unknown, what: string): unknown[] {
  if (!Array.isArray(v)) invalid(`${what} n'est pas une liste`)
  return v
}

function optionalNumber(v: unknown, what: string, fallback: number | null): number | null {
  return v === undefined || v === null ? fallback : asNumber(v, what)
}

function decodeCalibration(raw: unknown): ReliefCalibration {
  const o = asObject(raw, 'calibration')
  const calibration: ReliefCalibration = {
    metersPerPixel: asNumber(o.metersPerPixel, 'calibration.metersPerPixel'),
    originX: asNumber(o.originX, 'calibration.originX'),
    originY: asNumber(o.originY, 'calibration.originY'),
    widthPx: asNumber(o.widthPx, 'calibration.widthPx'),
    heightPx: asNumber(o.heightPx, 'calibration.heightPx'),
  }
  if (calibration.metersPerPixel <= 0) invalid('calibration.metersPerPixel doit être positif')
  if (!Number.isInteger(calibration.widthPx) || !Number.isInteger(calibration.heightPx)) {
    invalid('calibration : dimensions non entières')
  }
  if (calibration.widthPx <= 0 || calibration.heightPx <= 0) invalid('calibration : dimensions non positives')
  return calibration
}

function decodeHeightMeta(raw: unknown): ReliefHeightMeta {
  const o = asObject(raw, 'height')
  const meta: ReliefHeightMeta = {
    z0: asNumber(o.z0, 'height.z0'),
    step: asNumber(o.step, 'height.step'),
    low: asNumber(o.low, 'height.low'),
    high: asNumber(o.high, 'height.high'),
  }
  if (meta.step <= 0) invalid('height.step doit être positif')
  return meta
}

/** Sommets (centièmes entiers) et faces (indices) -> maillage à plat, avec la boîte de chaque face. */
function decodeMesh(rawVertices: unknown, rawFaces: unknown): FloorMesh {
  const flat = asArray(rawVertices ?? [], 'floors.vertices')
  if (flat.length % 3 !== 0) invalid('floors.vertices : longueur non multiple de 3')
  const vertices = new Float64Array(flat.length)
  for (let i = 0; i < flat.length; i++) {
    const c = asNumber(flat[i], 'floors.vertices[]')
    if (!Number.isInteger(c)) invalid('floors.vertices : centième non entier')
    vertices[i] = c / CENTI_PER_UNIT
  }
  const vertexCount = flat.length / 3
  const faces = asArray(rawFaces ?? [], 'floors.faces')
  const faceStart = new Uint32Array(faces.length + 1)
  const indices: number[] = []
  const bounds = new Float64Array(faces.length * 4)
  for (let f = 0; f < faces.length; f++) {
    const face = asArray(faces[f], 'floors.faces[]')
    if (face.length < 3) invalid('floors.faces : une face a moins de trois sommets')
    let minX = Infinity
    let minY = Infinity
    let maxX = -Infinity
    let maxY = -Infinity
    for (const idx of face) {
      const v = asNumber(idx, 'floors.faces[][]')
      if (!Number.isInteger(v) || v < 0 || v >= vertexCount) invalid(`floors.faces : indice ${v} hors des sommets`)
      indices.push(v)
      const x = vertices[3 * v]
      const y = vertices[3 * v + 1]
      if (x < minX) minX = x
      if (x > maxX) maxX = x
      if (y < minY) minY = y
      if (y > maxY) maxY = y
    }
    faceStart[f + 1] = indices.length
    bounds.set([minX, minY, maxX, maxY], f * 4)
  }
  return { vertices, faceStart, faceIndices: Uint32Array.from(indices), bounds }
}

function decodeFloors(raw: unknown): ReliefFloors {
  const o = asObject(raw ?? {}, 'floors')
  const levels = asArray(o.levels ?? [], 'floors.levels').map((v) => asNumber(v, 'floors.levels[]'))
  const share = optionalNumber(o.meshShare, 'floors.meshShare', null)
  if (share !== null && (share < 0 || share > 1)) invalid('floors.meshShare hors de 0..1')
  return {
    levels,
    min: optionalNumber(o.min, 'floors.min', null),
    max: optionalNumber(o.max, 'floors.max', null),
    merge: optionalNumber(o.merge, 'floors.merge', DEFAULT_FLOOR_MERGE_U) ?? DEFAULT_FLOOR_MERGE_U,
    tolerance:
      optionalNumber(o.tolerance, 'floors.tolerance', DEFAULT_FLOOR_TOLERANCE_U) ?? DEFAULT_FLOOR_TOLERANCE_U,
    meshShare: share,
    mesh: decodeMesh(o.vertices, o.faces),
  }
}

/**
 * decodePolylines décode des polylignes delta-encodées au dixième de pixel. Une polyligne d'un seul
 * sommet est écartée (rien à tracer) ; une longueur impaire ou un non-entier refuse le fichier.
 */
export function decodePolylines(raw: unknown, what: string): LineSet {
  const lines = asArray(raw ?? [], what)
  const builder = new LineSetBuilder()
  for (const line of lines) {
    const coords = asArray(line, `${what}[]`)
    if (coords.length % 2 !== 0) invalid(`${what} : polyligne de longueur impaire`)
    let u = 0
    let v = 0
    for (let i = 0; i < coords.length; i += 2) {
      const dx = asNumber(coords[i], `${what}[][]`)
      const dy = asNumber(coords[i + 1], `${what}[][]`)
      if (!Number.isInteger(dx) || !Number.isInteger(dy)) invalid(`${what} : dixième de pixel non entier`)
      u += dx
      v += dy
      builder.add(u / TENTHS_PER_PIXEL, v / TENTHS_PER_PIXEL)
    }
    builder.end()
  }
  return builder.build()
}

function decodeLines(raw: unknown): ReliefLines {
  const o = asObject(raw ?? {}, 'lines')
  return {
    steps: decodePolylines(o.steps, 'lines.steps'),
    contours: decodePolylines(o.contours, 'lines.contours'),
    outline: decodePolylines(o.outline, 'lines.outline'),
  }
}

function decodeDegradations(raw: unknown): ReliefDegradation[] {
  return asArray(raw ?? [], 'degradations').filter((code): code is ReliefDegradation =>
    KNOWN_DEGRADATIONS.includes(code as ReliefDegradation),
  )
}

/**
 * parseReliefData valide et décode le JSON d'un relief (déjà parsé par `JSON.parse` / `api.get`).
 * Lève `ReliefError('data-version')` pour un schéma inconnu, `ReliefError('data-invalid')` pour tout
 * autre manquement.
 */
export function parseReliefData(raw: unknown): ReliefData {
  const o = asObject(raw, 'racine')
  if (o.schemaVersion !== RELIEF_SCHEMA_VERSION) {
    throw new ReliefError('data-version', `schemaVersion ${String(o.schemaVersion)} non lue (attendue : ${RELIEF_SCHEMA_VERSION})`)
  }
  return {
    schemaVersion: RELIEF_SCHEMA_VERSION,
    calibration: decodeCalibration(o.calibration),
    height: decodeHeightMeta(o.height),
    floors: decodeFloors(o.floors),
    lines: decodeLines(o.lines),
    degradations: decodeDegradations(o.degradations),
  }
}

/** Un relief sans maillage : pas de commande d'étage, pas de coupe (D-7). */
export function hasNoMesh(data: ReliefData): boolean {
  return data.degradations.includes('sans-maillage') || data.floors.mesh.faceStart.length <= 1
}
